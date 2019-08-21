package framework

import (
	"fmt"
	"time"

	"github.com/appscode/go/crypto/rand"
	"github.com/appscode/go/encoding/json/types"
	. "github.com/onsi/gomega"
	policy "k8s.io/api/policy/v1beta1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	api "kubedb.dev/apimachinery/apis/kubedb/v1alpha1"
	"kubedb.dev/apimachinery/client/clientset/versioned/typed/kubedb/v1alpha1/util"
)

const (
	kindEviction = "Eviction"
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
			Version: types.StrYo(DBCatalogName),
		},
	}
}

func (f *Framework) CreateMemcached(obj *api.Memcached) error {
	_, err := f.dbClient.KubedbV1alpha1().Memcacheds(obj.Namespace).Create(obj)
	return err
}

func (f *Framework) GetMemcached(meta metav1.ObjectMeta) (*api.Memcached, error) {
	return f.dbClient.KubedbV1alpha1().Memcacheds(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
}

func (f *Framework) TryPatchMemcached(meta metav1.ObjectMeta, transform func(*api.Memcached) *api.Memcached) (*api.Memcached, error) {
	memcached, err := f.dbClient.KubedbV1alpha1().Memcacheds(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	memcached, _, err = util.PatchMemcached(f.dbClient.KubedbV1alpha1(), memcached, transform)
	return memcached, err
}

func (f *Framework) DeleteMemcached(meta metav1.ObjectMeta) error {
	return f.dbClient.KubedbV1alpha1().Memcacheds(meta.Namespace).Delete(meta.Name, deleteInForeground())
}

func (f *Framework) EventuallyMemcached(meta metav1.ObjectMeta) GomegaAsyncAssertion {
	return Eventually(
		func() bool {
			_, err := f.dbClient.KubedbV1alpha1().Memcacheds(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
			if err != nil {
				if kerr.IsNotFound(err) {
					return false
				}
				Expect(err).NotTo(HaveOccurred())
			}
			return true
		},
		time.Minute*13,
		time.Second*5,
	)
}

func (f *Framework) EventuallyMemcachedRunning(meta metav1.ObjectMeta) GomegaAsyncAssertion {
	return Eventually(
		func() bool {
			memcached, err := f.dbClient.KubedbV1alpha1().Memcacheds(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			return memcached.Status.Phase == api.DatabasePhaseRunning
		},
		time.Minute*5,
		time.Second*5,
	)
}

func (f *Framework) CleanMemcached() {
	memcachedList, err := f.dbClient.KubedbV1alpha1().Memcacheds(f.namespace).List(metav1.ListOptions{})
	if err != nil {
		return
	}
	for _, e := range memcachedList.Items {
		if _, _, err := util.PatchMemcached(f.dbClient.KubedbV1alpha1(), &e, func(in *api.Memcached) *api.Memcached {
			in.ObjectMeta.Finalizers = nil
			in.Spec.TerminationPolicy = api.TerminationPolicyWipeOut
			return in
		}); err != nil {
			fmt.Printf("error Patching Memcached. error: %v", err)
		}
	}
	if err := f.dbClient.KubedbV1alpha1().Memcacheds(f.namespace).DeleteCollection(deleteInForeground(), metav1.ListOptions{}); err != nil {
		fmt.Printf("error in deletion of Memcached. Error: %v", err)
	}
}

func (f *Framework) EvictPodsFromDeployment(meta metav1.ObjectMeta) error {
	var err error
	deployName := meta.Name
	//if PDB is not found, send error
	pdb, err := f.kubeClient.PolicyV1beta1().PodDisruptionBudgets(meta.Namespace).Get(deployName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	if pdb.Spec.MinAvailable == nil {
		return fmt.Errorf("found pdb %s spec.minAvailable nil", pdb.Name)
	}

	podSelector := labels.Set{
		api.LabelDatabaseKind: api.ResourceKindMemcached,
		api.LabelDatabaseName: meta.GetName(),
	}
	pods, err := f.kubeClient.CoreV1().Pods(meta.Namespace).List(metav1.ListOptions{LabelSelector: podSelector.String()})
	if err != nil {
		return err
	}
	podCount := len(pods.Items)
	if podCount < 1 {
		return fmt.Errorf("found no pod in namespace %s with given labels", meta.Namespace)
	}
	eviction := &policy.Eviction{
		TypeMeta: metav1.TypeMeta{
			APIVersion: policy.SchemeGroupVersion.String(),
			Kind:       kindEviction,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: meta.Namespace,
		},
		DeleteOptions: deleteInForeground(),
	}

	// try to evict as many pods as allowed in pdb
	minAvailable := pdb.Spec.MinAvailable.IntValue()
	for i, pod := range pods.Items {
		eviction.Name = pod.Name
		err = f.kubeClient.PolicyV1beta1().Evictions(eviction.Namespace).Evict(eviction)
		if i < (podCount - minAvailable) {
			if err != nil {
				return err
			}
		} else {
			// This pod should not get evicted
			if kerr.IsTooManyRequests(err) {
				err = nil
				break
			} else if err != nil {
				return err
			} else {
				return fmt.Errorf("expected pod %s/%s to be not evicted due to pdb %s", meta.Namespace, eviction.Name, pdb.Name)
			}
		}
	}
	return err
}
