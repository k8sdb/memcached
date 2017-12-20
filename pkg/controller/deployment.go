package controller

import (
	"fmt"
	"time"

	"github.com/appscode/go/types"
	app_util "github.com/appscode/kutil/apps/v1beta1"
	core_util "github.com/appscode/kutil/core/v1"
	api "github.com/kubedb/apimachinery/apis/kubedb/v1alpha1"
	"github.com/kubedb/apimachinery/client/typed/kubedb/v1alpha1/util"
	"github.com/kubedb/apimachinery/pkg/docker"
	"github.com/kubedb/apimachinery/pkg/eventer"
	apps "k8s.io/api/apps/v1beta1"
	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// Duration in Minute
	// Check whether pod under Deployment is running or not
	// Continue checking for this duration until failure
	durationCheckDeployment = time.Minute * 30
)

func (c *Controller) ensureDeployment(memcached *api.Memcached) error {
	if err := c.checkDeployment(memcached); err != nil {
		return err
	}

	deploymentMeta := metav1.ObjectMeta{
		Name:      memcached.OffshootName(),
		Namespace: memcached.Namespace,
	}

	replicas := memcached.Spec.Replicas
	if replicas < 1 {
		replicas = 1
	}

	if c.opt.EnableRbac {
		// Ensure ClusterRoles for database deployment
		if err := c.ensureRBACStuff(memcached); err != nil {
			return err
		}
	}

	deployment, err := app_util.CreateOrPatchDeployment(c.Client, deploymentMeta, func(in *apps.Deployment) *apps.Deployment {
		in = upsertObjectMeta(in, memcached)

		in.Spec.Replicas = types.Int32P(replicas)
		in.Spec.Template = core.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: in.ObjectMeta.Labels,
			},
		}

		in = upsertContainer(in, memcached)
		in = upsertDeploymentPort(in)

		in.Spec.Template.Spec.NodeSelector = memcached.Spec.NodeSelector
		in.Spec.Template.Spec.Affinity = memcached.Spec.Affinity
		in.Spec.Template.Spec.SchedulerName = memcached.Spec.SchedulerName
		in.Spec.Template.Spec.Tolerations = memcached.Spec.Tolerations

		in = upsertMonitoringContainer(in, memcached, c.opt.ExporterTag)

		if c.opt.EnableRbac {
			in.Spec.Template.Spec.ServiceAccountName = memcached.Name
		}

		return in
	})

	if err != nil {
		return err
	}

	// Check Deployment Pod status
	if err := c.CheckDeploymentPodStatus(deployment, durationCheckDeployment); err != nil {
		c.recorder.Eventf(
			memcached.ObjectReference(),
			core.EventTypeWarning,
			eventer.EventReasonFailedToStart,
			`Failed to CreateOrPatch Deployment. Reason: %v`,
			err,
		)
		return err
	} else {
		c.recorder.Event(
			memcached.ObjectReference(),
			core.EventTypeNormal,
			eventer.EventReasonSuccessfulCreate,
			"Successfully CreatedOrPatched Deployment",
		)
	}

	mg, err := util.PatchMemcached(c.ExtClient, memcached, func(in *api.Memcached) *api.Memcached {
		in.Status.Phase = api.DatabasePhaseRunning
		return in
	})
	if err != nil {
		c.recorder.Eventf(
			memcached,
			core.EventTypeWarning,
			eventer.EventReasonFailedToUpdate,
			err.Error(),
		)
		return err
	}
	*memcached = *mg
	return nil
}

func (c *Controller) checkDeployment(memcached *api.Memcached) error {
	// Deployment for Memcached database
	dbName := memcached.OffshootName()
	deployment, err := c.Client.AppsV1beta1().Deployments(memcached.Namespace).Get(dbName, metav1.GetOptions{})
	if err != nil {
		if kerr.IsNotFound(err) {
			return nil
		} else {
			return err
		}
	}

	if deployment.Labels[api.LabelDatabaseKind] != api.ResourceKindMemcached || deployment.Labels[api.LabelDatabaseName] != dbName {
		return fmt.Errorf(`intended deployment "%v" already exists`, dbName)
	}

	return nil
}

func upsertObjectMeta(deployment *apps.Deployment, memcached *api.Memcached) *apps.Deployment {
	deployment.Labels = core_util.UpsertMap(deployment.Labels, memcached.DeploymentLabels())
	deployment.Annotations = core_util.UpsertMap(deployment.Annotations, memcached.DeploymentAnnotations())
	return deployment
}

func upsertContainer(deployment *apps.Deployment, memcached *api.Memcached) *apps.Deployment {
	container := core.Container{
		Name:            api.ResourceNameMemcached,
		Image:           fmt.Sprintf("%s:%s", docker.ImageMemcached, memcached.Spec.Version),
		ImagePullPolicy: core.PullIfNotPresent,
		//todo: security context
	}
	containers := deployment.Spec.Template.Spec.Containers
	containers = core_util.UpsertContainer(containers, container)
	deployment.Spec.Template.Spec.Containers = containers
	return deployment
}

func upsertDeploymentPort(deployment *apps.Deployment) *apps.Deployment {
	getPorts := func() []core.ContainerPort {
		portList := []core.ContainerPort{
			{
				Name:          "db",
				ContainerPort: 11211,
				Protocol:      core.ProtocolTCP,
			},
		}
		return portList
	}

	for i, container := range deployment.Spec.Template.Spec.Containers {
		if container.Name == api.ResourceNameMemcached {
			deployment.Spec.Template.Spec.Containers[i].Ports = getPorts()
			return deployment
		}
	}
	return deployment
}

func upsertMonitoringContainer(deployment *apps.Deployment, memcached *api.Memcached, tag string) *apps.Deployment {
	if memcached.Spec.Monitor != nil &&
		memcached.Spec.Monitor.Agent == api.AgentCoreosPrometheus &&
		memcached.Spec.Monitor.Prometheus != nil {
		container := core.Container{
			Name: "exporter",
			Args: []string{
				"export",
				fmt.Sprintf("--address=:%d", memcached.Spec.Monitor.Prometheus.Port),
				"--v=3",
			},
			Image:           docker.ImageOperator + ":" + tag,
			ImagePullPolicy: core.PullIfNotPresent,
			Ports: []core.ContainerPort{
				{
					Name:          api.PrometheusExporterPortName,
					Protocol:      core.ProtocolTCP,
					ContainerPort: memcached.Spec.Monitor.Prometheus.Port,
				},
			},
		}
		containers := deployment.Spec.Template.Spec.Containers
		containers = core_util.UpsertContainer(containers, container)
		deployment.Spec.Template.Spec.Containers = containers
	}
	return deployment
}
