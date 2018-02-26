package validator

import (
	"fmt"

	"github.com/appscode/go/types"
	api "github.com/kubedb/apimachinery/apis/kubedb/v1alpha1"
	amv "github.com/kubedb/apimachinery/pkg/validator"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/kubernetes"
)

var (
	memcachedVersions = sets.NewString("1.5", "1.5.4")
)

func ValidateMemcached(client kubernetes.Interface, memcached *api.Memcached) error {
	if memcached.Spec.Version == "" {
		return fmt.Errorf(`object 'Version' is missing in '%v'`, memcached.Spec)
	}

	// check Memcached version validation
	if !memcachedVersions.Has(string(memcached.Spec.Version)) {
		return fmt.Errorf(`KubeDB doesn't support Memcached version: %s`, string(memcached.Spec.Version))
	}

	if memcached.Spec.Replicas != nil {
		replicas := types.Int32(memcached.Spec.Replicas)
		if replicas < 1 {
			return fmt.Errorf(`spec.replicas "%d" invalid`, replicas)
		}
	}

	monitorSpec := memcached.Spec.Monitor
	if monitorSpec != nil {
		if err := amv.ValidateMonitorSpec(monitorSpec); err != nil {
			return err
		}
	}
	return nil
}
