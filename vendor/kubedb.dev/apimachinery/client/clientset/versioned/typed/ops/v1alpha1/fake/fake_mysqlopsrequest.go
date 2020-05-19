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
	v1alpha1 "kubedb.dev/apimachinery/apis/ops/v1alpha1"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeMySQLOpsRequests implements MySQLOpsRequestInterface
type FakeMySQLOpsRequests struct {
	Fake *FakeOpsV1alpha1
	ns   string
}

var mysqlopsrequestsResource = schema.GroupVersionResource{Group: "ops.kubedb.com", Version: "v1alpha1", Resource: "mysqlopsrequests"}

var mysqlopsrequestsKind = schema.GroupVersionKind{Group: "ops.kubedb.com", Version: "v1alpha1", Kind: "MySQLOpsRequest"}

// Get takes name of the mySQLOpsRequest, and returns the corresponding mySQLOpsRequest object, and an error if there is any.
func (c *FakeMySQLOpsRequests) Get(name string, options v1.GetOptions) (result *v1alpha1.MySQLOpsRequest, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(mysqlopsrequestsResource, c.ns, name), &v1alpha1.MySQLOpsRequest{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.MySQLOpsRequest), err
}

// List takes label and field selectors, and returns the list of MySQLOpsRequests that match those selectors.
func (c *FakeMySQLOpsRequests) List(opts v1.ListOptions) (result *v1alpha1.MySQLOpsRequestList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(mysqlopsrequestsResource, mysqlopsrequestsKind, c.ns, opts), &v1alpha1.MySQLOpsRequestList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.MySQLOpsRequestList{ListMeta: obj.(*v1alpha1.MySQLOpsRequestList).ListMeta}
	for _, item := range obj.(*v1alpha1.MySQLOpsRequestList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested mySQLOpsRequests.
func (c *FakeMySQLOpsRequests) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(mysqlopsrequestsResource, c.ns, opts))

}

// Create takes the representation of a mySQLOpsRequest and creates it.  Returns the server's representation of the mySQLOpsRequest, and an error, if there is any.
func (c *FakeMySQLOpsRequests) Create(mySQLOpsRequest *v1alpha1.MySQLOpsRequest) (result *v1alpha1.MySQLOpsRequest, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(mysqlopsrequestsResource, c.ns, mySQLOpsRequest), &v1alpha1.MySQLOpsRequest{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.MySQLOpsRequest), err
}

// Update takes the representation of a mySQLOpsRequest and updates it. Returns the server's representation of the mySQLOpsRequest, and an error, if there is any.
func (c *FakeMySQLOpsRequests) Update(mySQLOpsRequest *v1alpha1.MySQLOpsRequest) (result *v1alpha1.MySQLOpsRequest, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(mysqlopsrequestsResource, c.ns, mySQLOpsRequest), &v1alpha1.MySQLOpsRequest{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.MySQLOpsRequest), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeMySQLOpsRequests) UpdateStatus(mySQLOpsRequest *v1alpha1.MySQLOpsRequest) (*v1alpha1.MySQLOpsRequest, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(mysqlopsrequestsResource, "status", c.ns, mySQLOpsRequest), &v1alpha1.MySQLOpsRequest{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.MySQLOpsRequest), err
}

// Delete takes name of the mySQLOpsRequest and deletes it. Returns an error if one occurs.
func (c *FakeMySQLOpsRequests) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(mysqlopsrequestsResource, c.ns, name), &v1alpha1.MySQLOpsRequest{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeMySQLOpsRequests) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(mysqlopsrequestsResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &v1alpha1.MySQLOpsRequestList{})
	return err
}

// Patch applies the patch and returns the patched mySQLOpsRequest.
func (c *FakeMySQLOpsRequests) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.MySQLOpsRequest, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(mysqlopsrequestsResource, c.ns, name, pt, data, subresources...), &v1alpha1.MySQLOpsRequest{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.MySQLOpsRequest), err
}