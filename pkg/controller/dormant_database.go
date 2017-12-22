package controller

import (
	"github.com/appscode/go/log"
	api "github.com/kubedb/apimachinery/apis/kubedb/v1alpha1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (c *Controller) Exists(om *metav1.ObjectMeta) (bool, error) {
	memcached, err := c.ExtClient.Memcacheds(om.Namespace).Get(om.Name, metav1.GetOptions{})
	if err != nil {
		if !kerr.IsNotFound(err) {
			return false, err
		}
		return false, nil
	}

	if memcached.DeletionTimestamp != nil {
		return false, nil
	}
	return true, nil
}

func (c *Controller) PauseDatabase(dormantDb *api.DormantDatabase) error {
	// Delete Service
	if err := c.DeleteService(dormantDb.Name, dormantDb.Namespace); err != nil {
		log.Errorln(err)
		return err
	}

	if err := c.deleteDeployment(dormantDb.OffshootName(), dormantDb.Namespace); err != nil && !kerr.IsNotFound(err) {
		log.Errorln(err)
		return err
	}

	memcached := &api.Memcached{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dormantDb.OffshootName(),
			Namespace: dormantDb.Namespace,
		},
	}
	if err := c.deleteRBACStuff(memcached); err != nil {
		log.Errorln(err)
		return err
	}
	return nil
}

func (c *Controller) WipeOutDatabase(dormantDb *api.DormantDatabase) error {
	log.Info("No snapshot for Memcached.")

	return nil
}

func (c *Controller) ResumeDatabase(dormantDb *api.DormantDatabase) error {
	origin := dormantDb.Spec.Origin
	objectMeta := origin.ObjectMeta

	memcached := &api.Memcached{
		ObjectMeta: metav1.ObjectMeta{
			Name:        objectMeta.Name,
			Namespace:   objectMeta.Namespace,
			Labels:      objectMeta.Labels,
			Annotations: objectMeta.Annotations,
		},
		Spec: *origin.Spec.Memcached,
	}

	if memcached.Annotations == nil {
		memcached.Annotations = make(map[string]string)
	}

	for key, val := range dormantDb.Annotations {
		memcached.Annotations[key] = val
	}

	_, err := c.ExtClient.Memcacheds(memcached.Namespace).Create(memcached)
	return err
}

func (c *Controller) createDormantDatabase(memcached *api.Memcached) (*api.DormantDatabase, error) {
	dormantDb := &api.DormantDatabase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      memcached.Name,
			Namespace: memcached.Namespace,
			Labels: map[string]string{
				api.LabelDatabaseKind: api.ResourceKindMemcached,
			},
		},
		Spec: api.DormantDatabaseSpec{
			Origin: api.Origin{
				ObjectMeta: metav1.ObjectMeta{
					Name:        memcached.Name,
					Namespace:   memcached.Namespace,
					Labels:      memcached.Labels,
					Annotations: memcached.Annotations,
				},
				Spec: api.OriginSpec{
					Memcached: &memcached.Spec,
				},
			},
		},
	}
	return c.ExtClient.DormantDatabases(dormantDb.Namespace).Create(dormantDb)
}
