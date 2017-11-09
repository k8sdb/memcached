package controller

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/appscode/log"
	tapi "github.com/k8sdb/apimachinery/apis/kubedb/v1alpha1"
	kutildb "github.com/k8sdb/apimachinery/client/typed/kubedb/v1alpha1/util"
	"github.com/k8sdb/apimachinery/pkg/eventer"
	"github.com/k8sdb/apimachinery/pkg/storage"
	"github.com/k8sdb/memcached/pkg/validator"
	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TODO: Use your resource instead of *tapi.Memcached
func (c *Controller) create(memcached *tapi.Memcached) error {
	// TODO: Use correct TryPatch method
	_, err := kutildb.TryPatchMemcached(c.ExtClient, memcached.ObjectMeta, func(in *tapi.Memcached) *tapi.Memcached {
		t := metav1.Now()
		in.Status.CreationTime = &t
		in.Status.Phase = tapi.DatabasePhaseCreating
		return in
	})

	if err != nil {
		c.recorder.Eventf(memcached.ObjectReference(), core.EventTypeWarning, eventer.EventReasonFailedToUpdate, err.Error())
		return err
	}

	if err := validator.ValidateMemcached(c.Client, memcached); err != nil {
		c.recorder.Event(memcached.ObjectReference(), core.EventTypeWarning, eventer.EventReasonInvalid, err.Error())
		return err
	}
	// Event for successful validation
	c.recorder.Event(
		memcached.ObjectReference(),
		core.EventTypeNormal,
		eventer.EventReasonSuccessfulValidate,
		"Successfully validate Memcached",
	)

	// Check DormantDatabase
	matched, err := c.matchDormantDatabase(memcached)
	if err != nil {
		return err
	}
	if matched {
		//TODO: Use Annotation Key
		memcached.Annotations = map[string]string{
			"kubedb.com/ignore": "",
		}
		if err := c.ExtClient.Memcacheds(memcached.Namespace).Delete(memcached.Name, &metav1.DeleteOptions{}); err != nil {
			return fmt.Errorf(
				`Failed to resume Memcached "%v" from DormantDatabase "%v". Error: %v`,
				memcached.Name,
				memcached.Name,
				err,
			)
		}

		_, err := kutildb.TryPatchDormantDatabase(c.ExtClient, memcached.ObjectMeta, func(in *tapi.DormantDatabase) *tapi.DormantDatabase {
			in.Spec.Resume = true
			return in
		})
		if err != nil {
			c.recorder.Eventf(memcached.ObjectReference(), core.EventTypeWarning, eventer.EventReasonFailedToUpdate, err.Error())
			return err
		}

		return nil
	}

	// Event for notification that kubernetes objects are creating
	c.recorder.Event(memcached.ObjectReference(), core.EventTypeNormal, eventer.EventReasonCreating, "Creating Kubernetes objects")

	// create Governing Service
	governingService := c.opt.GoverningService
	if err := c.CreateGoverningService(governingService, memcached.Namespace); err != nil {
		c.recorder.Eventf(
			memcached.ObjectReference(),
			core.EventTypeWarning,
			eventer.EventReasonFailedToCreate,
			`Failed to create Service: "%v". Reason: %v`,
			governingService,
			err,
		)
		return err
	}

	// ensure database Service
	if err := c.ensureService(memcached); err != nil {
		return err
	}

	// ensure database StatefulSet
	if err := c.ensureStatefulSet(memcached); err != nil {
		return err
	}

	c.recorder.Event(
		memcached.ObjectReference(),
		core.EventTypeNormal,
		eventer.EventReasonSuccessfulCreate,
		"Successfully created Memcached",
	)

	// Ensure Schedule backup
	c.ensureBackupScheduler(memcached)

	if memcached.Spec.Monitor != nil {
		if err := c.addMonitor(memcached); err != nil {
			c.recorder.Eventf(
				memcached.ObjectReference(),
				core.EventTypeWarning,
				eventer.EventReasonFailedToCreate,
				"Failed to add monitoring system. Reason: %v",
				err,
			)
			log.Errorln(err)
			return nil
		}
		c.recorder.Event(
			memcached.ObjectReference(),
			core.EventTypeNormal,
			eventer.EventReasonSuccessfulCreate,
			"Successfully added monitoring system.",
		)
	}
	return nil
}

func (c *Controller) matchDormantDatabase(memcached *tapi.Memcached) (bool, error) {
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

	var sendEvent = func(message string) (bool, error) {
		c.recorder.Event(
			memcached.ObjectReference(),
			core.EventTypeWarning,
			eventer.EventReasonFailedToCreate,
			message,
		)
		return false, errors.New(message)
	}

	// Check DatabaseKind
	// TODO: Change tapi.ResourceKindMemcached
	if dormantDb.Labels[tapi.LabelDatabaseKind] != tapi.ResourceKindMemcached {
		return sendEvent(fmt.Sprintf(`Invalid Memcached: "%v". Exists DormantDatabase "%v" of different Kind`,
			memcached.Name, dormantDb.Name))
	}

	// Check InitSpec
	// TODO: Change tapi.MemcachedInitSpec
	initSpecAnnotationStr := dormantDb.Annotations[tapi.MemcachedInitSpec]
	if initSpecAnnotationStr != "" {
		var initSpecAnnotation *tapi.InitSpec
		if err := json.Unmarshal([]byte(initSpecAnnotationStr), &initSpecAnnotation); err != nil {
			return sendEvent(err.Error())
		}

		if memcached.Spec.Init != nil {
			if !reflect.DeepEqual(initSpecAnnotation, memcached.Spec.Init) {
				return sendEvent("InitSpec mismatches with DormantDatabase annotation")
			}
		}
	}

	// Check Origin Spec
	drmnOriginSpec := dormantDb.Spec.Origin.Spec.Memcached
	originalSpec := memcached.Spec
	originalSpec.Init = nil

	// ---> Start
	// TODO: Use following part if database secret is supported
	// Otherwise, remove it
	if originalSpec.DatabaseSecret == nil {
		originalSpec.DatabaseSecret = &core.SecretVolumeSource{
			SecretName: memcached.Name + "-admin-auth",
		}
	}
	// ---> End

	if !reflect.DeepEqual(drmnOriginSpec, &originalSpec) {
		return sendEvent("Memcached spec mismatches with OriginSpec in DormantDatabases")
	}

	return true, nil
}

func (c *Controller) ensureService(memcached *tapi.Memcached) error {
	// Check if service name exists
	found, err := c.findService(memcached)
	if err != nil {
		return err
	}
	if found {
		return nil
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

func (c *Controller) ensureStatefulSet(memcached *tapi.Memcached) error {
	found, err := c.findStatefulSet(memcached)
	if err != nil {
		return err
	}
	if found {
		return nil
	}

	// Create statefulSet for Memcached database
	statefulSet, err := c.createStatefulSet(memcached)
	if err != nil {
		c.recorder.Eventf(
			memcached.ObjectReference(),
			core.EventTypeWarning,
			eventer.EventReasonFailedToCreate,
			"Failed to create StatefulSet. Reason: %v",
			err,
		)
		return err
	}

	// Check StatefulSet Pod status
	if err := c.CheckStatefulSetPodStatus(statefulSet, durationCheckStatefulSet); err != nil {
		c.recorder.Eventf(
			memcached.ObjectReference(),
			core.EventTypeWarning,
			eventer.EventReasonFailedToStart,
			`Failed to create StatefulSet. Reason: %v`,
			err,
		)
		return err
	} else {
		c.recorder.Event(
			memcached.ObjectReference(),
			core.EventTypeNormal,
			eventer.EventReasonSuccessfulCreate,
			"Successfully created StatefulSet",
		)
	}

	if memcached.Spec.Init != nil && memcached.Spec.Init.SnapshotSource != nil {
		// TODO: Use correct TryPatch method
		_, err := kutildb.TryPatchMemcached(c.ExtClient, memcached.ObjectMeta, func(in *tapi.Memcached) *tapi.Memcached {
			in.Status.Phase = tapi.DatabasePhaseInitializing
			return in
		})
		if err != nil {
			c.recorder.Eventf(memcached, core.EventTypeWarning, eventer.EventReasonFailedToUpdate, err.Error())
			return err
		}

		if err := c.initialize(memcached); err != nil {
			c.recorder.Eventf(
				memcached.ObjectReference(),
				core.EventTypeWarning,
				eventer.EventReasonFailedToInitialize,
				"Failed to initialize. Reason: %v",
				err,
			)
		}
	}

	// TODO: Use correct TryPatch method
	_, err = kutildb.TryPatchMemcached(c.ExtClient, memcached.ObjectMeta, func(in *tapi.Memcached) *tapi.Memcached {
		in.Status.Phase = tapi.DatabasePhaseRunning
		return in
	})
	if err != nil {
		c.recorder.Eventf(memcached, core.EventTypeWarning, eventer.EventReasonFailedToUpdate, err.Error())
		return err
	}
	return nil
}

func (c *Controller) ensureBackupScheduler(memcached *tapi.Memcached) {
	// Setup Schedule backup
	if memcached.Spec.BackupSchedule != nil {
		err := c.cronController.ScheduleBackup(memcached, memcached.ObjectMeta, memcached.Spec.BackupSchedule)
		if err != nil {
			c.recorder.Eventf(
				memcached.ObjectReference(),
				core.EventTypeWarning,
				eventer.EventReasonFailedToSchedule,
				"Failed to schedule snapshot. Reason: %v",
				err,
			)
			log.Errorln(err)
		}
	} else {
		c.cronController.StopBackupScheduling(memcached.ObjectMeta)
	}
}

const (
	durationCheckRestoreJob = time.Minute * 30
)

func (c *Controller) initialize(memcached *tapi.Memcached) error {
	snapshotSource := memcached.Spec.Init.SnapshotSource
	// Event for notification that kubernetes objects are creating
	c.recorder.Eventf(
		memcached.ObjectReference(),
		core.EventTypeNormal,
		eventer.EventReasonInitializing,
		`Initializing from Snapshot: "%v"`,
		snapshotSource.Name,
	)

	namespace := snapshotSource.Namespace
	if namespace == "" {
		namespace = memcached.Namespace
	}
	snapshot, err := c.ExtClient.Snapshots(namespace).Get(snapshotSource.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	secret, err := storage.NewOSMSecret(c.Client, snapshot)
	if err != nil {
		return err
	}
	_, err = c.Client.CoreV1().Secrets(secret.Namespace).Create(secret)
	if err != nil {
		return err
	}

	job, err := c.createRestoreJob(memcached, snapshot)
	if err != nil {
		return err
	}

	jobSuccess := c.CheckDatabaseRestoreJob(job, memcached, c.recorder, durationCheckRestoreJob)
	if jobSuccess {
		c.recorder.Event(
			memcached.ObjectReference(),
			core.EventTypeNormal,
			eventer.EventReasonSuccessfulInitialize,
			"Successfully completed initialization",
		)
	} else {
		c.recorder.Event(
			memcached.ObjectReference(),
			core.EventTypeWarning,
			eventer.EventReasonFailedToInitialize,
			"Failed to complete initialization",
		)
	}
	return nil
}

func (c *Controller) pause(memcached *tapi.Memcached) error {
	if memcached.Annotations != nil {
		if val, found := memcached.Annotations["kubedb.com/ignore"]; found {
			//TODO: Add Event Reason "Ignored"
			c.recorder.Event(memcached.ObjectReference(), core.EventTypeNormal, "Ignored", val)
			return nil
		}
	}

	c.recorder.Event(memcached.ObjectReference(), core.EventTypeNormal, eventer.EventReasonPausing, "Pausing Memcached")

	if memcached.Spec.DoNotPause {
		c.recorder.Eventf(
			memcached.ObjectReference(),
			core.EventTypeWarning,
			eventer.EventReasonFailedToPause,
			`Memcached "%v" is locked.`,
			memcached.Name,
		)

		if err := c.reCreateMemcached(memcached); err != nil {
			c.recorder.Eventf(
				memcached.ObjectReference(),
				core.EventTypeWarning,
				eventer.EventReasonFailedToCreate,
				`Failed to recreate Memcached: "%v". Reason: %v`,
				memcached.Name,
				err,
			)
			return err
		}
		return nil
	}

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

	c.cronController.StopBackupScheduling(memcached.ObjectMeta)

	if memcached.Spec.Monitor != nil {
		if err := c.deleteMonitor(memcached); err != nil {
			c.recorder.Eventf(
				memcached.ObjectReference(),
				core.EventTypeWarning,
				eventer.EventReasonFailedToDelete,
				"Failed to delete monitoring system. Reason: %v",
				err,
			)
			log.Errorln(err)
			return nil
		}
		c.recorder.Event(
			memcached.ObjectReference(),
			core.EventTypeNormal,
			eventer.EventReasonSuccessfulMonitorDelete,
			"Successfully deleted monitoring system.",
		)
	}
	return nil
}

func (c *Controller) update(oldMemcached, updatedMemcached *tapi.Memcached) error {
	if err := validator.ValidateMemcached(c.Client, updatedMemcached); err != nil {
		c.recorder.Event(updatedMemcached.ObjectReference(), core.EventTypeWarning, eventer.EventReasonInvalid, err.Error())
		return err
	}
	// Event for successful validation
	c.recorder.Event(
		updatedMemcached.ObjectReference(),
		core.EventTypeNormal,
		eventer.EventReasonSuccessfulValidate,
		"Successfully validate Memcached",
	)

	if err := c.ensureService(updatedMemcached); err != nil {
		return err
	}
	if err := c.ensureStatefulSet(updatedMemcached); err != nil {
		return err
	}

	if !reflect.DeepEqual(updatedMemcached.Spec.BackupSchedule, oldMemcached.Spec.BackupSchedule) {
		c.ensureBackupScheduler(updatedMemcached)
	}

	if !reflect.DeepEqual(oldMemcached.Spec.Monitor, updatedMemcached.Spec.Monitor) {
		if err := c.updateMonitor(oldMemcached, updatedMemcached); err != nil {
			c.recorder.Eventf(
				updatedMemcached.ObjectReference(),
				core.EventTypeWarning,
				eventer.EventReasonFailedToUpdate,
				"Failed to update monitoring system. Reason: %v",
				err,
			)
			log.Errorln(err)
			return nil
		}
		c.recorder.Event(
			updatedMemcached.ObjectReference(),
			core.EventTypeNormal,
			eventer.EventReasonSuccessfulMonitorUpdate,
			"Successfully updated monitoring system.",
		)

	}
	return nil
}
