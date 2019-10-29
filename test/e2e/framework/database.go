/*
Copyright The KubeDB Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package framework

import (
	"fmt"
	"strings"
	"time"

	api "kubedb.dev/apimachinery/apis/kubedb/v1alpha1"
	"kubedb.dev/memcached/pkg/controller"

	"github.com/bradfitz/gomemcache/memcache"
	. "github.com/onsi/gomega"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"kmodules.xyz/client-go/tools/portforward"
)

func (f *Framework) GetDatabasePod(meta metav1.ObjectMeta) (*core.Pod, error) {
	pods, err := f.kubeClient.CoreV1().Pods(meta.Namespace).List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, pod := range pods.Items {
		if strings.HasPrefix(pod.Name, meta.Name) {
			return &pod, nil
		}
	}
	return nil, fmt.Errorf("no pod found for memcache: %s", meta.Name)
}

func (f *Framework) GetMemcacheClient(meta metav1.ObjectMeta) (*memcache.Client, error) {
	memcached, err := f.GetMemcached(meta)
	if err != nil {
		return nil, err
	}

	clientPod, err := f.GetDatabasePod(meta)
	if err != nil {
		return nil, err
	}

	f.tunnel = portforward.NewTunnel(
		f.kubeClient.CoreV1().RESTClient(),
		f.restConfig,
		memcached.Namespace,
		clientPod.Name,
		11211,
	)

	if err := f.tunnel.ForwardPort(); err != nil {
		return nil, err
	}

	mc := memcache.New(fmt.Sprintf("localhost:%v", f.tunnel.Local))
	mc.Timeout = time.Second * 5 // Increase the client's timeout to 5 seconds
	return mc, nil
}

func (f *Framework) EventuallySetItem(meta metav1.ObjectMeta, key, value string) GomegaAsyncAssertion {
	return Eventually(
		func() bool {
			client, err := f.GetMemcacheClient(meta)
			Expect(err).NotTo(HaveOccurred())

			defer f.tunnel.Close()

			err = client.Set(&memcache.Item{Key: key, Value: []byte(value)})
			return err == nil
		},
		time.Minute*5,
		time.Second*5,
	)
}

func (f *Framework) EventuallyGetItem(meta metav1.ObjectMeta, key string) GomegaAsyncAssertion {
	return Eventually(
		func() string {
			client, err := f.GetMemcacheClient(meta)
			Expect(err).NotTo(HaveOccurred())

			defer f.tunnel.Close()

			item, err := client.Get(key)
			if err != nil {
				return ""
			}
			return string(item.Value)
		},
		time.Minute*5,
		time.Second*5,
	)
}

func (f *Invocation) EventuallyConfigSourceVolumeMounted(meta metav1.ObjectMeta) GomegaAsyncAssertion {

	return Eventually(
		func() bool {
			pod, err := f.GetDatabasePod(meta)
			if err != nil {
				return false
			}

			for _, c := range pod.Spec.Containers {
				if c.Name == api.ResourceSingularMemcached {
					for _, vm := range c.VolumeMounts {
						if vm.Name == controller.CONFIG_SOURCE_VOLUME {
							return true
						}
					}
				}
			}
			return false
		},
		time.Minute*5,
		time.Second*5,
	)
}

func (f *Framework) EventuallyMemcachedConfigs(meta metav1.ObjectMeta) GomegaAsyncAssertion {

	return Eventually(
		func() string {

			// TODO
			ret := make([]string, 0)
			return strings.Join(ret, " ")
		},
		time.Minute*5,
		time.Second*5,
	)
}
