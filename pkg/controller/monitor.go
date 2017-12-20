package controller

import (
	"fmt"

	"github.com/appscode/kutil/tools/monitoring/agents"
	kapi "github.com/appscode/kutil/tools/monitoring/api"
	mona "github.com/appscode/kutil/tools/monitoring/api"
	api "github.com/kubedb/apimachinery/apis/kubedb/v1alpha1"
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

func (c *Controller) addOrUpdateMonitor(memcached *api.Memcached) error {
	agent, err := c.newMonitorController(memcached)
	if err != nil {
		return err
	}
	return agent.Add(memcached.StatsAccessor(), memcached.Spec.Monitor)
}

func (c *Controller) deleteMonitor(memcached *api.Memcached) error {
	agent, err := c.newMonitorController(memcached)
	if err != nil {
		return err
	}
	return agent.Delete(memcached.StatsAccessor(), memcached.Spec.Monitor)
}

func (c *Controller) manageMonitor(memcached *api.Memcached) error {
	if memcached.Spec.Monitor != nil {
		return c.addOrUpdateMonitor(memcached)
	} else {
		agent := agents.New(kapi.AgentCoreOSPrometheus, c.Client, c.ApiExtKubeClient, c.promClient)
		return agent.Add(memcached.StatsAccessor(), memcached.Spec.Monitor)
	}
	return nil
}
