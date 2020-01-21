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
	v1alpha1 "kubedb.dev/apimachinery/apis/dba/v1alpha1"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakePgBouncerModificationRequests implements PgBouncerModificationRequestInterface
type FakePgBouncerModificationRequests struct {
	Fake *FakeDbaV1alpha1
}

var pgbouncermodificationrequestsResource = schema.GroupVersionResource{Group: "dba.kubedb.com", Version: "v1alpha1", Resource: "pgbouncermodificationrequests"}

var pgbouncermodificationrequestsKind = schema.GroupVersionKind{Group: "dba.kubedb.com", Version: "v1alpha1", Kind: "PgBouncerModificationRequest"}

// Get takes name of the pgBouncerModificationRequest, and returns the corresponding pgBouncerModificationRequest object, and an error if there is any.
func (c *FakePgBouncerModificationRequests) Get(name string, options v1.GetOptions) (result *v1alpha1.PgBouncerModificationRequest, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootGetAction(pgbouncermodificationrequestsResource, name), &v1alpha1.PgBouncerModificationRequest{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.PgBouncerModificationRequest), err
}

// List takes label and field selectors, and returns the list of PgBouncerModificationRequests that match those selectors.
func (c *FakePgBouncerModificationRequests) List(opts v1.ListOptions) (result *v1alpha1.PgBouncerModificationRequestList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootListAction(pgbouncermodificationrequestsResource, pgbouncermodificationrequestsKind, opts), &v1alpha1.PgBouncerModificationRequestList{})
	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.PgBouncerModificationRequestList{ListMeta: obj.(*v1alpha1.PgBouncerModificationRequestList).ListMeta}
	for _, item := range obj.(*v1alpha1.PgBouncerModificationRequestList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested pgBouncerModificationRequests.
func (c *FakePgBouncerModificationRequests) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewRootWatchAction(pgbouncermodificationrequestsResource, opts))
}

// Create takes the representation of a pgBouncerModificationRequest and creates it.  Returns the server's representation of the pgBouncerModificationRequest, and an error, if there is any.
func (c *FakePgBouncerModificationRequests) Create(pgBouncerModificationRequest *v1alpha1.PgBouncerModificationRequest) (result *v1alpha1.PgBouncerModificationRequest, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootCreateAction(pgbouncermodificationrequestsResource, pgBouncerModificationRequest), &v1alpha1.PgBouncerModificationRequest{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.PgBouncerModificationRequest), err
}

// Update takes the representation of a pgBouncerModificationRequest and updates it. Returns the server's representation of the pgBouncerModificationRequest, and an error, if there is any.
func (c *FakePgBouncerModificationRequests) Update(pgBouncerModificationRequest *v1alpha1.PgBouncerModificationRequest) (result *v1alpha1.PgBouncerModificationRequest, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateAction(pgbouncermodificationrequestsResource, pgBouncerModificationRequest), &v1alpha1.PgBouncerModificationRequest{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.PgBouncerModificationRequest), err
}

// Delete takes name of the pgBouncerModificationRequest and deletes it. Returns an error if one occurs.
func (c *FakePgBouncerModificationRequests) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewRootDeleteAction(pgbouncermodificationrequestsResource, name), &v1alpha1.PgBouncerModificationRequest{})
	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakePgBouncerModificationRequests) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewRootDeleteCollectionAction(pgbouncermodificationrequestsResource, listOptions)

	_, err := c.Fake.Invokes(action, &v1alpha1.PgBouncerModificationRequestList{})
	return err
}

// Patch applies the patch and returns the patched pgBouncerModificationRequest.
func (c *FakePgBouncerModificationRequests) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.PgBouncerModificationRequest, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(pgbouncermodificationrequestsResource, name, pt, data, subresources...), &v1alpha1.PgBouncerModificationRequest{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.PgBouncerModificationRequest), err
}