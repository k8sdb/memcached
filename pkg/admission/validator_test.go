package admission

import (
	"net/http"
	"testing"

	"github.com/appscode/go/types"
	"github.com/appscode/kutil/meta"
	catalogapi "github.com/kubedb/apimachinery/apis/catalog/v1alpha1"
	dbapi "github.com/kubedb/apimachinery/apis/kubedb/v1alpha1"
	extFake "github.com/kubedb/apimachinery/client/clientset/versioned/fake"
	"github.com/kubedb/apimachinery/client/clientset/versioned/scheme"
	admission "k8s.io/api/admission/v1beta1"
	apps "k8s.io/api/apps/v1"
	authenticationV1 "k8s.io/api/authentication/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	clientSetScheme "k8s.io/client-go/kubernetes/scheme"
	mona "kmodules.xyz/monitoring-agent-api/api/v1"
)

func init() {
	scheme.AddToScheme(clientSetScheme.Scheme)
}

var requestKind = metaV1.GroupVersionKind{
	Group:   dbapi.SchemeGroupVersion.Group,
	Version: dbapi.SchemeGroupVersion.Version,
	Kind:    dbapi.ResourceKindMemcached,
}

func TestMemcachedValidator_Admit(t *testing.T) {
	for _, c := range cases {
		t.Run(c.testName, func(t *testing.T) {
			validator := MemcachedValidator{}

			validator.initialized = true
			validator.extClient = extFake.NewSimpleClientset(
				&catalogapi.MemcachedVersion{
					ObjectMeta: metaV1.ObjectMeta{
						Name: "1.5.4",
					},
				},
			)
			validator.client = fake.NewSimpleClientset()

			objJS, err := meta.MarshalToJson(&c.object, dbapi.SchemeGroupVersion)
			if err != nil {
				panic(err)
			}
			oldObjJS, err := meta.MarshalToJson(&c.oldObject, dbapi.SchemeGroupVersion)
			if err != nil {
				panic(err)
			}

			req := new(admission.AdmissionRequest)

			req.Kind = c.kind
			req.Name = c.objectName
			req.Namespace = c.namespace
			req.Operation = c.operation
			req.UserInfo = authenticationV1.UserInfo{}
			req.Object.Raw = objJS
			req.OldObject.Raw = oldObjJS

			if c.heatUp {
				if _, err := validator.extClient.KubedbV1alpha1().Memcacheds(c.namespace).Create(&c.object); err != nil && !kerr.IsAlreadyExists(err) {
					t.Errorf(err.Error())
				}
			}
			if c.operation == admission.Delete {
				req.Object = runtime.RawExtension{}
			}
			if c.operation != admission.Update {
				req.OldObject = runtime.RawExtension{}
			}

			response := validator.Admit(req)
			if c.result == true {
				if response.Allowed != true {
					t.Errorf("expected: 'Allowed=true'. but got response: %v", response)
				}
			} else if c.result == false {
				if response.Allowed == true || response.Result.Code == http.StatusInternalServerError {
					t.Errorf("expected: 'Allowed=false', but got response: %v", response)
				}
			}
		})
	}

}

var cases = []struct {
	testName   string
	kind       metaV1.GroupVersionKind
	objectName string
	namespace  string
	operation  admission.Operation
	object     dbapi.Memcached
	oldObject  dbapi.Memcached
	heatUp     bool
	result     bool
}{
	{"Create Valid Memcached",
		requestKind,
		"foo",
		"default",
		admission.Create,
		sampleMemcached(),
		dbapi.Memcached{},
		false,
		true,
	},
	{"Create Invalid Memcached",
		requestKind,
		"foo",
		"default",
		admission.Create,
		getAwkwardMemcached(),
		dbapi.Memcached{},
		false,
		false,
	},
	{"Edit Memcached Spec.Version",
		requestKind,
		"foo",
		"default",
		admission.Update,
		editSpecVersion(sampleMemcached()),
		sampleMemcached(),
		false,
		false,
	},
	{"Edit Status",
		requestKind,
		"foo",
		"default",
		admission.Update,
		editStatus(sampleMemcached()),
		sampleMemcached(),
		false,
		true,
	},
	{"Edit Spec.Monitor",
		requestKind,
		"foo",
		"default",
		admission.Update,
		editSpecMonitor(sampleMemcached()),
		sampleMemcached(),
		false,
		true,
	},
	{"Edit Invalid Spec.Monitor",
		requestKind,
		"foo",
		"default",
		admission.Update,
		editSpecInvalidMonitor(sampleMemcached()),
		sampleMemcached(),
		false,
		false,
	},
	{"Edit Spec.DoNotPause",
		requestKind,
		"foo",
		"default",
		admission.Update,
		editSpecDoNotPause(sampleMemcached()),
		sampleMemcached(),
		false,
		true,
	},
	{"Delete Memcached when Spec.DoNotPause=true",
		requestKind,
		"foo",
		"default",
		admission.Delete,
		sampleMemcached(),
		dbapi.Memcached{},
		true,
		false,
	},
	{"Delete Memcached when Spec.DoNotPause=false",
		requestKind,
		"foo",
		"default",
		admission.Delete,
		editSpecDoNotPause(sampleMemcached()),
		dbapi.Memcached{},
		true,
		true,
	},
	{"Delete Non Existing Memcached",
		requestKind,
		"foo",
		"default",
		admission.Delete,
		dbapi.Memcached{},
		dbapi.Memcached{},
		false,
		true,
	},
}

func sampleMemcached() dbapi.Memcached {
	return dbapi.Memcached{
		TypeMeta: metaV1.TypeMeta{
			Kind:       dbapi.ResourceKindMemcached,
			APIVersion: dbapi.SchemeGroupVersion.String(),
		},
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "foo",
			Namespace: "default",
			Labels: map[string]string{
				dbapi.LabelDatabaseKind: dbapi.ResourceKindMemcached,
			},
		},
		Spec: dbapi.MemcachedSpec{
			Version:    "1.5.4",
			Replicas:   types.Int32P(3),
			DoNotPause: true,
			UpdateStrategy: apps.DeploymentStrategy{
				Type: apps.RollingUpdateStatefulSetStrategyType,
			},
			TerminationPolicy: dbapi.TerminationPolicyPause,
		},
	}
}

func getAwkwardMemcached() dbapi.Memcached {
	memcached := sampleMemcached()
	memcached.Spec.Version = "3.0"
	return memcached
}

func editSpecVersion(old dbapi.Memcached) dbapi.Memcached {
	old.Spec.Version = "1.5.3"
	return old
}

func editStatus(old dbapi.Memcached) dbapi.Memcached {
	old.Status = dbapi.MemcachedStatus{
		Phase: dbapi.DatabasePhaseCreating,
	}
	return old
}

func editSpecMonitor(old dbapi.Memcached) dbapi.Memcached {
	old.Spec.Monitor = &mona.AgentSpec{
		Agent: mona.AgentPrometheusBuiltin,
		Prometheus: &mona.PrometheusSpec{
			Port: 5670,
		},
	}
	return old
}

// should be failed because more fields required for COreOS Monitoring
func editSpecInvalidMonitor(old dbapi.Memcached) dbapi.Memcached {
	old.Spec.Monitor = &mona.AgentSpec{
		Agent: mona.AgentCoreOSPrometheus,
	}
	return old
}

func editSpecDoNotPause(old dbapi.Memcached) dbapi.Memcached {
	old.Spec.DoNotPause = false
	return old
}
