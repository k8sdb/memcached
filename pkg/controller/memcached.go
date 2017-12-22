package controller

import (
	"fmt"
	"reflect"

	"github.com/appscode/go/log"
	"github.com/appscode/kutil"
	api "github.com/kubedb/apimachinery/apis/kubedb/v1alpha1"
	"github.com/kubedb/apimachinery/client/typed/kubedb/v1alpha1/util"
	"github.com/kubedb/apimachinery/pkg/eventer"
	"github.com/kubedb/memcached/pkg/validator"
	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (c *Controller) create(memcached *api.Memcached) error {
	if err := validator.ValidateMemcached(c.Client, memcached); err != nil {
		c.recorder.Event(
			memcached.ObjectReference(),
			core.EventTypeWarning,
			eventer.EventReasonInvalid,
			err.Error(),
		)
		return nil // user error so just record error and don't retry.
	}

	if memcached.Status.CreationTime == nil {
		mc, _, err := util.PatchMemcached(c.ExtClient, memcached, func(in *api.Memcached) *api.Memcached {
			t := metav1.Now()
			in.Status.CreationTime = &t
			in.Status.Phase = api.DatabasePhaseCreating
			return in
		})
		if err != nil {
			c.recorder.Eventf(
				memcached.ObjectReference(),
				core.EventTypeWarning,
				eventer.EventReasonFailedToUpdate,
				err.Error(),
			)
			return err
		}
		memcached.Status = mc.Status
	}

	// Dynamic Defaulting
	// Assign Default Monitoring Port
	if err := c.setMonitoringPort(memcached); err != nil {
		return err
	}
	// set replica to at least 1
	if memcached.Spec.Replicas < 1 {
		memcached.Spec.Replicas = 1
	}

	// Check DormantDatabase
	if matched, err := c.matchDormantDatabase(memcached); err != nil || matched {
		return err
	}

	// ensure database Service
	ok1, er1 := c.ensureService(memcached)
	if er1 != nil {
		return er1
	}

	// ensure database Deployment
	ok2, er2 := c.ensureDeployment(memcached)
	if er2 != nil {
		return er2
	}

	if ok1 == kutil.VerbCreated && ok2 == kutil.VerbCreated {
		c.recorder.Event(
			memcached.ObjectReference(),
			core.EventTypeNormal,
			eventer.EventReasonSuccessful,
			"Successfully created Memcached",
		)
	} else if ok1 == kutil.VerbPatched || ok2 == kutil.VerbPatched {
		c.recorder.Event(
			memcached.ObjectReference(),
			core.EventTypeNormal,
			eventer.EventReasonSuccessful,
			"Successfully patched Memcached",
		)
	}

	if err := c.manageMonitor(memcached); err != nil {
		c.recorder.Eventf(
			memcached.ObjectReference(),
			core.EventTypeWarning,
			eventer.EventReasonFailedToCreate,
			"Failed to manage monitoring system. Reason: %v",
			err,
		)
		log.Errorln(err)
		return nil
	}
	return nil
}

func (c *Controller) setMonitoringPort(memcached *api.Memcached) error {
	if memcached.Spec.Monitor != nil &&
		memcached.Spec.Monitor.Prometheus != nil {
		if memcached.Spec.Monitor.Prometheus.Port == 0 {
			mc, _, err := util.PatchMemcached(c.ExtClient, memcached, func(in *api.Memcached) *api.Memcached {
				in.Spec.Monitor.Prometheus.Port = api.PrometheusExporterPortNumber
				return in
			})

			if err != nil {
				c.recorder.Eventf(
					memcached.ObjectReference(),
					core.EventTypeWarning,
					eventer.EventReasonFailedToUpdate,
					err.Error(),
				)
				return err
			}
			memcached.Spec = mc.Spec
		}
	}
	return nil
}

func (c *Controller) matchDormantDatabase(memcached *api.Memcached) (bool, error) {
	// Check if DormantDatabase exists or not
	dormantDb, err := c.ExtClient.DormantDatabases(memcached.Namespace).Get(memcached.Name, metav1.GetOptions{})
	if err != nil {
		if !kerr.IsNotFound(err) {
			c.recorder.Eventf(
				memcached.ObjectReference(),
				core.EventTypeWarning,
				eventer.EventReasonFailedToGet,
				`Fail to get DormantDatabase: "%v". Reason: %v`,
				memcached.Name,
				err,
			)
			return false, err
		}
		return false, nil
	}

	var sendEvent = func(message string, args ...interface{}) (bool, error) {
		c.recorder.Eventf(
			memcached.ObjectReference(),
			core.EventTypeWarning,
			eventer.EventReasonFailedToCreate,
			message,
			args,
		)
		return false, fmt.Errorf(message, args)
	}

	// Check DatabaseKind
	if dormantDb.Labels[api.LabelDatabaseKind] != api.ResourceKindMemcached {
		return sendEvent(fmt.Sprintf(`Invalid Memcached: "%v". Exists DormantDatabase "%v" of different Kind`, memcached.Name, dormantDb.Name))
	}

	// Check Origin Spec
	drmnOriginSpec := dormantDb.Spec.Origin.Spec.Memcached
	originalSpec := memcached.Spec

	if !reflect.DeepEqual(drmnOriginSpec, &originalSpec) {
		return sendEvent("Memcached spec mismatches with OriginSpec in DormantDatabases")
	}

	if err := c.ExtClient.Memcacheds(memcached.Namespace).Delete(memcached.Name, &metav1.DeleteOptions{}); err != nil {
		return sendEvent(`failed to resume Memcached "%v" from DormantDatabase "%v". Error: %v`, memcached.Name, memcached.Name, err)
	}

	_, _, err = util.PatchDormantDatabase(c.ExtClient, dormantDb, func(in *api.DormantDatabase) *api.DormantDatabase {
		in.Spec.Resume = true
		return in
	})
	if err != nil {
		c.recorder.Eventf(
			memcached.ObjectReference(),
			core.EventTypeWarning,
			eventer.EventReasonFailedToUpdate,
			err.Error(),
		)
		return sendEvent(err.Error())
	}
	return true, nil
}

func (c *Controller) pause(memcached *api.Memcached) error {
	//if memcached.Spec.DoNotPause {
	//	c.recorder.Eventf(
	//		memcached.ObjectReference(),
	//		core.EventTypeWarning,
	//		eventer.EventReasonFailedToPause,
	//		`Memcached "%v" is locked.`,
	//		memcached.Name,
	//	)
	//
	//	if err := c.reCreateMemcached(memcached); err != nil {
	//		c.recorder.Eventf(
	//			memcached.ObjectReference(),
	//			core.EventTypeWarning,
	//			eventer.EventReasonFailedToCreate,
	//			`Failed to recreate Memcached: "%v". Reason: %v`,
	//			memcached.Name,
	//			err,
	//		)
	//		return err
	//	}
	//	return nil
	//}

	if _, err := c.createDormantDatabase(memcached); err != nil {
		c.recorder.Eventf(
			memcached.ObjectReference(),
			core.EventTypeWarning,
			eventer.EventReasonFailedToCreate,
			`Failed to create DormantDatabase: "%v". Reason: %v`,
			memcached.Name,
			err,
		)
		return err
	}
	c.recorder.Eventf(
		memcached.ObjectReference(),
		core.EventTypeNormal,
		eventer.EventReasonSuccessfulCreate,
		`Successfully created DormantDatabase: "%v"`,
		memcached.Name,
	)

	if memcached.Spec.Monitor != nil {
		if vt, err := c.deleteMonitor(memcached); err != nil {
			c.recorder.Eventf(
				memcached.ObjectReference(),
				core.EventTypeWarning,
				eventer.EventReasonFailedToDelete,
				"Failed to delete monitoring system. Reason: %v",
				err,
			)
			log.Errorln(err)
			return nil
		} else if vt != kutil.VerbUnchanged {
			c.recorder.Eventf(
				memcached.ObjectReference(),
				core.EventTypeNormal,
				eventer.EventReasonSuccessfulMonitorDelete,
				"Successfully %s monitoring system.",
				vt,
			)
		}
	}
	return nil
}

//func (c *Controller) reCreateMemcached(memcached *api.Memcached) error {
//	//reCreateMemcached
//	return nil
//}
