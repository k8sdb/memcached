/*
Copyright AppsCode Inc. and Contributors

Licensed under the AppsCode Community License 1.0.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://github.com/appscode/licenses/raw/1.0.0/AppsCode-Community-1.0.0.md

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package framework

import (
	"context"
	"fmt"

	api "kubedb.dev/apimachinery/apis/kubedb/v1alpha2"
	"kubedb.dev/apimachinery/client/clientset/versioned/typed/kubedb/v1alpha2/util"

	"github.com/appscode/go/crypto/rand"
	. "github.com/onsi/gomega"
	policy "k8s.io/api/policy/v1beta1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	meta_util "kmodules.xyz/client-go/meta"
)

const (
	kindEviction = "Eviction"
)

func (fi *Invocation) Memcached() *api.Memcached {
	return &api.Memcached{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix("memcached"),
			Namespace: fi.namespace,
		},
		Spec: api.MemcachedSpec{
			Version:           DBCatalogName,
			TerminationPolicy: api.TerminationPolicyHalt,
		},
	}
}

func (f *Framework) CreateMemcached(obj *api.Memcached) error {
	_, err := f.dbClient.KubedbV1alpha2().Memcacheds(obj.Namespace).Create(context.TODO(), obj, metav1.CreateOptions{})
	return err
}

func (f *Framework) GetMemcached(meta metav1.ObjectMeta) (*api.Memcached, error) {
	return f.dbClient.KubedbV1alpha2().Memcacheds(meta.Namespace).Get(context.TODO(), meta.Name, metav1.GetOptions{})
}

func (f *Framework) PatchMemcached(meta metav1.ObjectMeta, transform func(*api.Memcached) *api.Memcached) (*api.Memcached, error) {
	memcached, err := f.dbClient.KubedbV1alpha2().Memcacheds(meta.Namespace).Get(context.TODO(), meta.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	memcached, _, err = util.PatchMemcached(context.TODO(), f.dbClient.KubedbV1alpha2(), memcached, transform, metav1.PatchOptions{})
	return memcached, err
}

func (f *Framework) DeleteMemcached(meta metav1.ObjectMeta) error {
	err := f.dbClient.KubedbV1alpha2().Memcacheds(meta.Namespace).Delete(context.TODO(), meta.Name, meta_util.DeleteInForeground())
	if !kerr.IsNotFound(err) {
		return err
	}
	return nil
}

func (f *Framework) EventuallyMemcached(meta metav1.ObjectMeta) GomegaAsyncAssertion {
	return Eventually(
		func() bool {
			_, err := f.dbClient.KubedbV1alpha2().Memcacheds(meta.Namespace).Get(context.TODO(), meta.Name, metav1.GetOptions{})
			if err != nil {
				if kerr.IsNotFound(err) {
					return false
				}
				Expect(err).NotTo(HaveOccurred())
			}
			return true
		},
		Timeout,
		RetryInterval,
	)
}

func (f *Framework) EventuallyMemcachedPhase(meta metav1.ObjectMeta) GomegaAsyncAssertion {
	return Eventually(
		func() api.DatabasePhase {
			db, err := f.dbClient.KubedbV1alpha2().Memcacheds(meta.Namespace).Get(context.TODO(), meta.Name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			return db.Status.Phase
		},
		Timeout,
		RetryInterval,
	)
}

func (f *Framework) EventuallyMemcachedRunning(meta metav1.ObjectMeta) GomegaAsyncAssertion {
	return Eventually(
		func() bool {
			memcached, err := f.dbClient.KubedbV1alpha2().Memcacheds(meta.Namespace).Get(context.TODO(), meta.Name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			return memcached.Status.Phase == api.DatabasePhaseReady
		},
		Timeout,
		RetryInterval,
	)
}

func (f *Framework) CleanMemcached() {
	memcachedList, err := f.dbClient.KubedbV1alpha2().Memcacheds(f.namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return
	}
	for _, e := range memcachedList.Items {
		if _, _, err := util.PatchMemcached(context.TODO(), f.dbClient.KubedbV1alpha2(), &e, func(in *api.Memcached) *api.Memcached {
			in.ObjectMeta.Finalizers = nil
			in.Spec.TerminationPolicy = api.TerminationPolicyWipeOut
			return in
		}, metav1.PatchOptions{}); err != nil {
			fmt.Printf("error Patching Memcached. error: %v", err)
		}
	}
	if err := f.dbClient.KubedbV1alpha2().Memcacheds(f.namespace).DeleteCollection(context.TODO(), meta_util.DeleteInForeground(), metav1.ListOptions{}); err != nil {
		fmt.Printf("error in deletion of Memcached. Error: %v", err)
	}
}

func (f *Framework) EvictPodsFromDeployment(meta metav1.ObjectMeta) error {
	var err error
	deployName := meta.Name
	//if PDB is not found, send error
	pdb, err := f.kubeClient.PolicyV1beta1().PodDisruptionBudgets(meta.Namespace).Get(context.TODO(), deployName, metav1.GetOptions{})
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
	pods, err := f.kubeClient.CoreV1().Pods(meta.Namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: podSelector.String()})
	if err != nil {
		return err
	}
	podCount := len(pods.Items)
	if podCount < 1 {
		return fmt.Errorf("found no pod in namespace %s with given labels", meta.Namespace)
	}
	foreground := meta_util.DeleteInForeground()
	eviction := &policy.Eviction{
		TypeMeta: metav1.TypeMeta{
			APIVersion: policy.SchemeGroupVersion.String(),
			Kind:       kindEviction,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: meta.Namespace,
		},
		DeleteOptions: &foreground,
	}

	// try to evict as many pods as allowed in pdb
	minAvailable := pdb.Spec.MinAvailable.IntValue()
	for i, pod := range pods.Items {
		eviction.Name = pod.Name
		err = f.kubeClient.PolicyV1beta1().Evictions(eviction.Namespace).Evict(context.TODO(), eviction)
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
