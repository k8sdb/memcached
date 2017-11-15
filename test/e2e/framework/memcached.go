package framework

import (
	"time"

	"github.com/appscode/go/crypto/rand"
	"github.com/appscode/go/encoding/json/types"
	tapi "github.com/k8sdb/apimachinery/apis/kubedb/v1alpha1"
	kutildb "github.com/k8sdb/apimachinery/client/typed/kubedb/v1alpha1/util"
	. "github.com/onsi/gomega"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (f *Invocation) Memcached() *tapi.Memcached {
	return &tapi.Memcached{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix("memcached"),
			Namespace: f.namespace,
			Labels: map[string]string{
				"app": f.app,
			},
		},
		Spec: tapi.MemcachedSpec{
			Version: types.StrYo("1"),
		},
	}
}

func (f *Framework) CreateMemcached(obj *tapi.Memcached) error {
	_, err := f.extClient.Memcacheds(obj.Namespace).Create(obj)
	return err
}

func (f *Framework) GetMemcached(meta metav1.ObjectMeta) (*tapi.Memcached, error) {
	return f.extClient.Memcacheds(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
}

func (f *Framework) TryPatchMemcached(meta metav1.ObjectMeta, transform func(*tapi.Memcached) *tapi.Memcached) (*tapi.Memcached, error) {
	return kutildb.TryPatchMemcached(f.extClient, meta, transform)
}

func (f *Framework) DeleteMemcached(meta metav1.ObjectMeta) error {
	return f.extClient.Memcacheds(meta.Namespace).Delete(meta.Name, &metav1.DeleteOptions{})
}

func (f *Framework) EventuallyMemcached(meta metav1.ObjectMeta) GomegaAsyncAssertion {
	return Eventually(
		func() bool {
			_, err := f.extClient.Memcacheds(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
			if err != nil {
				if kerr.IsNotFound(err) {
					return false
				} else {
					Expect(err).NotTo(HaveOccurred())
				}
			}
			return true
		},
		time.Minute*5,
		time.Second*5,
	)
}

func (f *Framework) EventuallyMemcachedRunning(meta metav1.ObjectMeta) GomegaAsyncAssertion {
	return Eventually(
		func() bool {
			memcached, err := f.extClient.Memcacheds(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			return memcached.Status.Phase == tapi.DatabasePhaseRunning
		},
		time.Minute*5,
		time.Second*5,
	)
}
