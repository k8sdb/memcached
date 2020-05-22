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

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	"context"

	v1alpha1 "kubedb.dev/apimachinery/apis/catalog/v1alpha1"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeMySQLVersions implements MySQLVersionInterface
type FakeMySQLVersions struct {
	Fake *FakeCatalogV1alpha1
}

var mysqlversionsResource = schema.GroupVersionResource{Group: "catalog.kubedb.com", Version: "v1alpha1", Resource: "mysqlversions"}

var mysqlversionsKind = schema.GroupVersionKind{Group: "catalog.kubedb.com", Version: "v1alpha1", Kind: "MySQLVersion"}

// Get takes name of the mySQLVersion, and returns the corresponding mySQLVersion object, and an error if there is any.
func (c *FakeMySQLVersions) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.MySQLVersion, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootGetAction(mysqlversionsResource, name), &v1alpha1.MySQLVersion{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.MySQLVersion), err
}

// List takes label and field selectors, and returns the list of MySQLVersions that match those selectors.
func (c *FakeMySQLVersions) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.MySQLVersionList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootListAction(mysqlversionsResource, mysqlversionsKind, opts), &v1alpha1.MySQLVersionList{})
	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.MySQLVersionList{ListMeta: obj.(*v1alpha1.MySQLVersionList).ListMeta}
	for _, item := range obj.(*v1alpha1.MySQLVersionList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested mySQLVersions.
func (c *FakeMySQLVersions) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewRootWatchAction(mysqlversionsResource, opts))
}

// Create takes the representation of a mySQLVersion and creates it.  Returns the server's representation of the mySQLVersion, and an error, if there is any.
func (c *FakeMySQLVersions) Create(ctx context.Context, mySQLVersion *v1alpha1.MySQLVersion, opts v1.CreateOptions) (result *v1alpha1.MySQLVersion, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootCreateAction(mysqlversionsResource, mySQLVersion), &v1alpha1.MySQLVersion{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.MySQLVersion), err
}

// Update takes the representation of a mySQLVersion and updates it. Returns the server's representation of the mySQLVersion, and an error, if there is any.
func (c *FakeMySQLVersions) Update(ctx context.Context, mySQLVersion *v1alpha1.MySQLVersion, opts v1.UpdateOptions) (result *v1alpha1.MySQLVersion, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateAction(mysqlversionsResource, mySQLVersion), &v1alpha1.MySQLVersion{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.MySQLVersion), err
}

// Delete takes name of the mySQLVersion and deletes it. Returns an error if one occurs.
func (c *FakeMySQLVersions) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewRootDeleteAction(mysqlversionsResource, name), &v1alpha1.MySQLVersion{})
	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeMySQLVersions) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewRootDeleteCollectionAction(mysqlversionsResource, listOpts)

	_, err := c.Fake.Invokes(action, &v1alpha1.MySQLVersionList{})
	return err
}

// Patch applies the patch and returns the patched mySQLVersion.
func (c *FakeMySQLVersions) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.MySQLVersion, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(mysqlversionsResource, name, pt, data, subresources...), &v1alpha1.MySQLVersion{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.MySQLVersion), err
}
