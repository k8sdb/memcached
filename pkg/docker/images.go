package docker

import (
	api "github.com/kubedb/apimachinery/apis/kubedb/v1alpha1"
)

const (
	ImageKubedbOperator    = "operator"
	ImageMemcachedOperator = "ms-operator"
	ImageMemcached         = "memcached"
	ImageMemcachedTools    = "memcached-tools"
)

type Docker struct {
	// Docker Registry
	Registry string
	// Exporter tag
	ExporterTag string
}

func (d Docker) GetImage(memcached *api.Memcached) string {
	return d.Registry + "/" + ImageMemcached
}

func (d Docker) GetImageWithTag(memcached *api.Memcached) string {
	return d.GetImage(memcached) + ":" + string(memcached.Spec.Version)
}

func (d Docker) GetOperatorImage(memcached *api.Memcached) string {
	return d.Registry + "/" + ImageKubedbOperator
}

func (d Docker) GetOperatorImageWithTag(memcached *api.Memcached) string {
	return d.GetOperatorImage(memcached) + ":" + d.ExporterTag
}

func (d Docker) GetToolsImage(memcached *api.Memcached) string {
	return d.Registry + "/" + ImageMemcachedTools
}

func (d Docker) GetToolsImageWithTag(memcached *api.Memcached) string {
	return d.GetToolsImage(memcached) + ":" + string(memcached.Spec.Version)
}
