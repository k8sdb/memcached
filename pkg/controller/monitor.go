package controller

import (
	"fmt"

	"github.com/appscode/kube-mon/agents"
	mona "github.com/appscode/kube-mon/api"
	"github.com/appscode/kutil"
	api "github.com/kubedb/apimachinery/apis/kubedb/v1alpha1"
	"github.com/kubedb/apimachinery/pkg/eventer"
	core "k8s.io/api/core/v1"
)

func (c *Controller) newMonitorController(memcached *api.Memcached) (mona.Agent, error) {
	monitorSpec := memcached.Spec.Monitor

	if monitorSpec == nil {
		return nil, fmt.Errorf("MonitorSpec not found in %v", memcached.Spec)
	}

	if monitorSpec.Prometheus != nil {
		return agents.New(monitorSpec.Agent, c.Client, c.ApiExtKubeClient, c.promClient), nil
	}

	return nil, fmt.Errorf("monitoring controller not found for %v", monitorSpec)
}

func (c *Controller) addOrUpdateMonitor(memcached *api.Memcached) (kutil.VerbType, error) {
	agent, err := c.newMonitorController(memcached)
	if err != nil {
		return kutil.VerbUnchanged, err
	}
	return agent.CreateOrUpdate(memcached.StatsAccessor(), memcached.Spec.Monitor)
}

func (c *Controller) deleteMonitor(memcached *api.Memcached) (kutil.VerbType, error) {
	agent, err := c.newMonitorController(memcached)
	if err != nil {
		return kutil.VerbUnchanged, err
	}
	return agent.Delete(memcached.StatsAccessor())
}

// todo: needs to set on status
func (c *Controller) manageMonitor(memcached *api.Memcached) error {
	vt := kutil.VerbUnchanged
	if memcached.Spec.Monitor != nil {
		ok1, err := c.addOrUpdateMonitor(memcached)
		if err != nil {
			return err
		}
		vt = ok1
	} else {
		agent := agents.New(mona.AgentCoreOSPrometheus, c.Client, c.ApiExtKubeClient, c.promClient)
		ok1, err := agent.CreateOrUpdate(memcached.StatsAccessor(), memcached.Spec.Monitor)
		if err != nil {
			return err
		}
		vt = ok1
	}
	if vt != kutil.VerbUnchanged {
		c.recorder.Eventf(
			memcached.ObjectReference(),
			core.EventTypeNormal,
			eventer.EventReasonSuccessful,
			"Successfully %v monitoring system.",
			vt,
		)
	}
	return nil
}
