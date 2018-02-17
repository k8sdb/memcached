package framework

import (
	"time"

	"github.com/appscode/go/crypto/rand"
	"github.com/appscode/go/encoding/json/types"
	core_util "github.com/appscode/kutil/core/v1"
	api "github.com/kubedb/apimachinery/apis/kubedb/v1alpha1"
	"github.com/kubedb/apimachinery/client/clientset/versioned/typed/kubedb/v1alpha1/util"
	. "github.com/onsi/gomega"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (f *Invocation) Memcached() *api.Memcached {
	return &api.Memcached{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix("memcached"),
			Namespace: f.namespace,
			Labels: map[string]string{
				"app": f.app,
			},
		},
		Spec: api.MemcachedSpec{
			Version: types.StrYo("1.5.4-alpine"),
		},
	}
}

func (f *Framework) CreateMemcached(obj *api.Memcached) error {
	_, err := f.extClient.Memcacheds(obj.Namespace).Create(obj)
	return err
}

func (f *Framework) GetMemcached(meta metav1.ObjectMeta) (*api.Memcached, error) {
	return f.extClient.Memcacheds(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
}

func (f *Framework) TryPatchMemcached(meta metav1.ObjectMeta, transform func(*api.Memcached) *api.Memcached) (*api.Memcached, error) {
	memcached, err := f.extClient.Memcacheds(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	memcached, _, err = util.PatchMemcached(f.extClient, memcached, transform)
	return memcached, err
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
			return memcached.Status.Phase == api.DatabasePhaseRunning
		},
		time.Minute*5,
		time.Second*5,
	)
}

func (f *Framework) CleanMemcached() {
	memcachedList, err := f.extClient.Memcacheds(f.namespace).List(metav1.ListOptions{})
	if err != nil {
		return
	}
	for _, m := range memcachedList.Items {
		util.PatchMemcached(f.extClient, &m, func(in *api.Memcached) *api.Memcached {
			in.ObjectMeta = core_util.RemoveFinalizer(in.ObjectMeta, api.GenericKey)
			return in
		})
	}
}
