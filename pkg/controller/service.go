package controller

import (
	"fmt"

	core_util "github.com/appscode/kutil/core/v1"
	api "github.com/kubedb/apimachinery/apis/kubedb/v1alpha1"
	"github.com/kubedb/apimachinery/pkg/eventer"
	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func (c *Controller) ensureService(memcached *api.Memcached) error {
	// Check if service name exists
	if err := c.checkService(memcached); err != nil {
		return err
	}

	// create database Service
	if err := c.createService(memcached); err != nil {
		c.recorder.Eventf(
			memcached.ObjectReference(),
			core.EventTypeWarning,
			eventer.EventReasonFailedToCreate,
			"Failed to create Service. Reason: %v",
			err,
		)
		return err
	}
	return nil
}

func (c *Controller) checkService(memcached *api.Memcached) error {
	name := memcached.OffshootName()
	service, err := c.Client.CoreV1().Services(memcached.Namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		if kerr.IsNotFound(err) {
			return nil
		} else {
			return err
		}
	}

	if service.Spec.Selector[api.LabelDatabaseName] != name {
		return fmt.Errorf(`intended service "%v" already exists`, name)
	}

	return nil
}

func (c *Controller) createService(memcached *api.Memcached) error {
	meta := metav1.ObjectMeta{
		Name:      memcached.OffshootName(),
		Namespace: memcached.Namespace,
	}

	_, err := core_util.CreateOrPatchService(c.Client, meta, func(in *core.Service) *core.Service {

		in.Labels = memcached.OffshootLabels()
		in.Spec.Ports = upsertServicePort(in, memcached)
		in.Spec.Selector = memcached.OffshootLabels()

		return in
	})
	return err
}

func upsertServicePort(service *core.Service, memcached *api.Memcached) []core.ServicePort {
	desiredPorts := []core.ServicePort{
		{
			Name:       "db",
			Protocol:   core.ProtocolTCP,
			Port:       11211,
			TargetPort: intstr.FromString("db"),
		},
	}

	if memcached.Spec.Monitor != nil &&
		memcached.Spec.Monitor.Agent == api.AgentCoreosPrometheus &&
		memcached.Spec.Monitor.Prometheus != nil {
		desiredPorts = append(desiredPorts, core.ServicePort{
			Name:       api.PrometheusExporterPortName,
			Protocol:   core.ProtocolTCP,
			Port:       memcached.Spec.Monitor.Prometheus.Port,
			TargetPort: intstr.FromString(api.PrometheusExporterPortName),
		})
	}
	return updatePorts(service.Spec.Ports, desiredPorts)
}

func updatePorts(current, desired []core.ServicePort) []core.ServicePort {
	if current == nil {
		return desired
	}

	curPorts := make(map[int32]core.ServicePort)
	for _, p := range current {
		curPorts[p.Port] = p
	}

	for i, dp := range desired {
		cp, ok := curPorts[dp.Port]

		// svc port not found
		if !ok {
			continue
		}

		delete(curPorts, dp.Port)

		if dp.NodePort == 0 {
			dp.NodePort = cp.NodePort // avoid reassigning port
		}
		desired[i] = dp
	}
	return desired
}
