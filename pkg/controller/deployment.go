package controller

import (
	"fmt"
	"path/filepath"

	"github.com/appscode/go/types"
	mon_api "github.com/appscode/kube-mon/api"
	"github.com/appscode/kutil"
	app_util "github.com/appscode/kutil/apps/v1"
	core_util "github.com/appscode/kutil/core/v1"
	api "github.com/kubedb/apimachinery/apis/kubedb/v1alpha1"
	"github.com/kubedb/apimachinery/pkg/eventer"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientsetscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/reference"
)

const (
	PARSER_SCRIPT_NAME             = "config-parser.sh"
	PARSER_SCRIPT_VOLUME           = "config-parser"
	PARSER_SCRIPT_VOLUME_MOUNTPATH = "/usr/config-parser/"
	CONFIG_FILE_NAME               = "memcached.conf"
	CONFIG_SOURCE_VOLUME           = "custom-config"
	CONFIG_SOURCE_VOLUME_MOUNTPATH = "/usr/config/"
)

func (c *Controller) ensureDeployment(memcached *api.Memcached) (kutil.VerbType, error) {
	if err := c.checkDeployment(memcached); err != nil {
		return kutil.VerbUnchanged, err
	}

	// Create statefulSet for Memcached database
	deployment, vt, err := c.createDeployment(memcached)
	if err != nil {
		return kutil.VerbUnchanged, err
	}
	// Check Deployment Pod status
	if vt != kutil.VerbUnchanged {
		if err := app_util.WaitUntilDeploymentReady(c.Client, deployment.ObjectMeta); err != nil {
			if ref, rerr := reference.GetReference(clientsetscheme.Scheme, memcached); rerr == nil {
				c.recorder.Eventf(
					ref,
					core.EventTypeWarning,
					eventer.EventReasonFailedToStart,
					`Failed to CreateOrPatch StatefulSet. Reason: %v`,
					err,
				)
			}
			return kutil.VerbUnchanged, err
		}
		if ref, rerr := reference.GetReference(clientsetscheme.Scheme, memcached); rerr == nil {
			c.recorder.Eventf(
				ref,
				core.EventTypeNormal,
				eventer.EventReasonSuccessful,
				"Successfully %v StatefulSet",
				vt,
			)
		}
	}
	return vt, nil
}

func (c *Controller) checkDeployment(memcached *api.Memcached) error {
	// Deployment for Memcached database
	dbName := memcached.OffshootName()
	deployment, err := c.Client.AppsV1().Deployments(memcached.Namespace).Get(dbName, metav1.GetOptions{})
	if err != nil {
		if kerr.IsNotFound(err) {
			return nil
		}
		return err
	}
	if deployment.Labels[api.LabelDatabaseKind] != api.ResourceKindMemcached || deployment.Labels[api.LabelDatabaseName] != dbName {
		return fmt.Errorf(`intended deployment "%v" already exists`, dbName)
	}
	return nil
}

func (c *Controller) createDeployment(memcached *api.Memcached) (*apps.Deployment, kutil.VerbType, error) {
	deploymentMeta := metav1.ObjectMeta{
		Name:      memcached.OffshootName(),
		Namespace: memcached.Namespace,
	}

	ref, rerr := reference.GetReference(clientsetscheme.Scheme, memcached)
	if rerr != nil {
		return nil, kutil.VerbUnchanged, rerr
	}

	return app_util.CreateOrPatchDeployment(c.Client, deploymentMeta, func(in *apps.Deployment) *apps.Deployment {
		in.ObjectMeta = core_util.EnsureOwnerReference(in.ObjectMeta, ref)
		in.Labels = core_util.UpsertMap(in.Labels, memcached.DeploymentLabels())
		in.Annotations = core_util.UpsertMap(in.Annotations, memcached.DeploymentAnnotations())

		in.Spec.Replicas = memcached.Spec.Replicas
		in.Spec.Template.Labels = in.Labels
		in.Spec.Selector = &metav1.LabelSelector{
			MatchLabels: in.Labels,
		}

		in.Spec.Template.Spec.Containers = core_util.UpsertContainer(in.Spec.Template.Spec.Containers, core.Container{
			Name:            api.ResourceSingularMemcached,
			Image:           c.docker.GetImageWithTag(memcached),
			ImagePullPolicy: core.PullIfNotPresent,
			Ports: []core.ContainerPort{
				{
					Name:          "db",
					ContainerPort: 11211,
					Protocol:      core.ProtocolTCP,
				},
			},
			Resources: memcached.Spec.Resources,
		})
		if memcached.GetMonitoringVendor() == mon_api.VendorPrometheus {
			in.Spec.Template.Spec.Containers = core_util.UpsertContainer(in.Spec.Template.Spec.Containers, core.Container{
				Name: "exporter",
				Args: append([]string{
					"export",
					fmt.Sprintf("--address=:%d", memcached.Spec.Monitor.Prometheus.Port),
					fmt.Sprintf("--enable-analytics=%v", c.EnableAnalytics),
				}, c.LoggerOptions.ToFlags()...),
				Image:           c.docker.GetOperatorImageWithTag(memcached),
				ImagePullPolicy: core.PullIfNotPresent,
				Ports: []core.ContainerPort{
					{
						Name:          api.PrometheusExporterPortName,
						Protocol:      core.ProtocolTCP,
						ContainerPort: memcached.Spec.Monitor.Prometheus.Port,
					},
				},
			})
		}

		in.Spec.Template.Spec.NodeSelector = memcached.Spec.NodeSelector
		in.Spec.Template.Spec.Affinity = memcached.Spec.Affinity
		in.Spec.Template.Spec.SchedulerName = memcached.Spec.SchedulerName
		in.Spec.Template.Spec.Tolerations = memcached.Spec.Tolerations
		in.Spec.Template.Spec.ImagePullSecrets = memcached.Spec.ImagePullSecrets
		in = upsertUserEnv(in, memcached)
		in = upsertCustomConfig(in, memcached)
		return in
	})
}

// upsertUserEnv add/overwrite env from user provided env in crd spec
func upsertUserEnv(deployment *apps.Deployment, memcached *api.Memcached) *apps.Deployment {
	for i, container := range deployment.Spec.Template.Spec.Containers {
		if container.Name == api.ResourceSingularMemcached {
			deployment.Spec.Template.Spec.Containers[i].Env = core_util.UpsertEnvVars(container.Env, memcached.Spec.Env...)
			return deployment
		}
	}
	return deployment
}

// upsertCustomConfig insert custom configuration volume if provided.
// it also insert a script to parse config parameters from provided memcached.conf file.
func upsertCustomConfig(deployment *apps.Deployment, memcached *api.Memcached) *apps.Deployment {
	if memcached.Spec.ConfigSource != nil {
		for i, container := range deployment.Spec.Template.Spec.Containers {
			if container.Name == api.ResourceSingularMemcached {
				configParserVolumeMount := core.VolumeMount{
					Name:      PARSER_SCRIPT_VOLUME,
					MountPath: PARSER_SCRIPT_VOLUME_MOUNTPATH,
				}

				configSourceVolumeMount := core.VolumeMount{
					Name:      CONFIG_SOURCE_VOLUME,
					MountPath: CONFIG_SOURCE_VOLUME_MOUNTPATH,
				}

				volumeMounts := container.VolumeMounts
				volumeMounts = core_util.UpsertVolumeMount(volumeMounts, configParserVolumeMount)
				volumeMounts = core_util.UpsertVolumeMount(volumeMounts, configSourceVolumeMount)
				deployment.Spec.Template.Spec.Containers[i].VolumeMounts = volumeMounts

				configSourceVolume := core.Volume{
					Name:         CONFIG_SOURCE_VOLUME,
					VolumeSource: *memcached.Spec.ConfigSource,
				}

				configParserVolume := core.Volume{
					Name: PARSER_SCRIPT_VOLUME,
					VolumeSource: core.VolumeSource{
						ConfigMap: &core.ConfigMapVolumeSource{
							LocalObjectReference: core.LocalObjectReference{
								Name: getConfigParserName(memcached),
							},
							DefaultMode: types.Int32P(0744),
						},
					},
				}

				volumes := deployment.Spec.Template.Spec.Volumes
				volumes = core_util.UpsertVolume(volumes, configSourceVolume)
				volumes = core_util.UpsertVolume(volumes, configParserVolume)
				deployment.Spec.Template.Spec.Volumes = volumes

				// add command to run parser script
				// parser script will run docker-entrypoint.sh after parsing config file.
				deployment.Spec.Template.Spec.Containers[i].Command = []string{filepath.Join(PARSER_SCRIPT_VOLUME_MOUNTPATH, PARSER_SCRIPT_NAME)}
				break
			}
		}
	}

	return deployment
}
