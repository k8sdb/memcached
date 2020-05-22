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

package v1alpha1

import (
	"context"
	"time"

	v1alpha1 "kubedb.dev/apimachinery/apis/catalog/v1alpha1"
	scheme "kubedb.dev/apimachinery/client/clientset/versioned/scheme"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// MemcachedVersionsGetter has a method to return a MemcachedVersionInterface.
// A group's client should implement this interface.
type MemcachedVersionsGetter interface {
	MemcachedVersions() MemcachedVersionInterface
}

// MemcachedVersionInterface has methods to work with MemcachedVersion resources.
type MemcachedVersionInterface interface {
	Create(ctx context.Context, memcachedVersion *v1alpha1.MemcachedVersion, opts v1.CreateOptions) (*v1alpha1.MemcachedVersion, error)
	Update(ctx context.Context, memcachedVersion *v1alpha1.MemcachedVersion, opts v1.UpdateOptions) (*v1alpha1.MemcachedVersion, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*v1alpha1.MemcachedVersion, error)
	List(ctx context.Context, opts v1.ListOptions) (*v1alpha1.MemcachedVersionList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.MemcachedVersion, err error)
	MemcachedVersionExpansion
}

// memcachedVersions implements MemcachedVersionInterface
type memcachedVersions struct {
	client rest.Interface
}

// newMemcachedVersions returns a MemcachedVersions
func newMemcachedVersions(c *CatalogV1alpha1Client) *memcachedVersions {
	return &memcachedVersions{
		client: c.RESTClient(),
	}
}

// Get takes name of the memcachedVersion, and returns the corresponding memcachedVersion object, and an error if there is any.
func (c *memcachedVersions) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.MemcachedVersion, err error) {
	result = &v1alpha1.MemcachedVersion{}
	err = c.client.Get().
		Resource("memcachedversions").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of MemcachedVersions that match those selectors.
func (c *memcachedVersions) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.MemcachedVersionList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1alpha1.MemcachedVersionList{}
	err = c.client.Get().
		Resource("memcachedversions").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested memcachedVersions.
func (c *memcachedVersions) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Resource("memcachedversions").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a memcachedVersion and creates it.  Returns the server's representation of the memcachedVersion, and an error, if there is any.
func (c *memcachedVersions) Create(ctx context.Context, memcachedVersion *v1alpha1.MemcachedVersion, opts v1.CreateOptions) (result *v1alpha1.MemcachedVersion, err error) {
	result = &v1alpha1.MemcachedVersion{}
	err = c.client.Post().
		Resource("memcachedversions").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(memcachedVersion).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a memcachedVersion and updates it. Returns the server's representation of the memcachedVersion, and an error, if there is any.
func (c *memcachedVersions) Update(ctx context.Context, memcachedVersion *v1alpha1.MemcachedVersion, opts v1.UpdateOptions) (result *v1alpha1.MemcachedVersion, err error) {
	result = &v1alpha1.MemcachedVersion{}
	err = c.client.Put().
		Resource("memcachedversions").
		Name(memcachedVersion.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(memcachedVersion).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the memcachedVersion and deletes it. Returns an error if one occurs.
func (c *memcachedVersions) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	return c.client.Delete().
		Resource("memcachedversions").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *memcachedVersions) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Resource("memcachedversions").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched memcachedVersion.
func (c *memcachedVersions) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.MemcachedVersion, err error) {
	result = &v1alpha1.MemcachedVersion{}
	err = c.client.Patch(pt).
		Resource("memcachedversions").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
