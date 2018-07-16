package controller

import (
	"fmt"

	"github.com/appscode/kutil"
	core_util "github.com/appscode/kutil/core/v1"
	api "github.com/kubedb/apimachinery/apis/kubedb/v1alpha1"
	"github.com/kubedb/apimachinery/pkg/eventer"
	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientsetscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/reference"
)

// ensureConfigParser ensure that respective configMap exist with config file parsing script.
func (c *Controller) ensureConfigParser(memcached *api.Memcached) error {
	// check if config-parser configMap name exists
	if err := c.checkConfigParser(memcached); err != nil {
		return err
	}

	// create config-parser configMap
	vt, err := c.createConfigParser(memcached)
	if err != nil {
		if ref, rerr := reference.GetReference(clientsetscheme.Scheme, memcached); rerr == nil {
			c.recorder.Eventf(
				ref,
				core.EventTypeWarning,
				eventer.EventReasonFailedToCreate,
				"Failed to create configMap. Reason: %v",
				err,
			)
		}
		return err
	} else if vt != kutil.VerbUnchanged {
		if ref, rerr := reference.GetReference(clientsetscheme.Scheme, memcached); rerr == nil {
			c.recorder.Eventf(
				ref,
				core.EventTypeNormal,
				eventer.EventReasonSuccessful,
				"Successfully %s configMap",
				vt,
			)
		}
	}
	return nil
}

func (c *Controller) checkConfigParser(memcached *api.Memcached) error {
	name := getConfigParserName(memcached)
	configmap, err := c.Client.CoreV1().ConfigMaps(memcached.Namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		if kerr.IsNotFound(err) {
			return nil
		}
		return err
	}
	if configmap.Labels[api.LabelDatabaseName] != memcached.OffshootName() {
		return fmt.Errorf(`intended configMap "%v" already exists`, name)
	}
	return nil
}

func (c *Controller) createConfigParser(memcached *api.Memcached) (kutil.VerbType, error) {
	meta := metav1.ObjectMeta{
		Name:      getConfigParserName(memcached),
		Namespace: memcached.Namespace,
	}

	ref, rerr := reference.GetReference(clientsetscheme.Scheme, memcached)
	if rerr != nil {
		return kutil.VerbUnchanged, rerr
	}

	_, ok, err := core_util.CreateOrPatchConfigMap(c.Client, meta, func(in *core.ConfigMap) *core.ConfigMap {
		in.ObjectMeta = core_util.EnsureOwnerReference(in.ObjectMeta, ref)
		in.Labels = memcached.OffshootLabels()
		in.Data = map[string]string{
			PARSER_SCRIPT_NAME: getConfigParserScript(),
		}
		return in
	})
	return ok, err
}

func getConfigParserName(mc *api.Memcached) string {
	return mc.OffshootName() + "-config-parser"
}

func getConfigParserScript() string {
	return `
#!/bin/sh
set -e

cmd="docker-entrypoint.sh"
configFile="/usr/config/memcached.conf"

# parseConfig parse the config file and convert it to argument to pass to memcached binary
parseConfig() {
    args=""
    while read -r line || [ -n "$line" ]; do
        case $line in
            -*) # for config format -c 500 or --conn-limit=500
                args="$(echo "${args}" "${line}")"
            ;;
            [a-z]*) # for config format conn-limit = 500
                trimmedLine="$(echo "${line}" | tr -d '[:space:]')" # trim all spaces from the line (i.e conn-limit=500)
                param="$(echo "--${trimmedLine}")"                  # append -- in front of trimmedLine (i.e --conn-limit=500)
                args="$(echo "${args}" "${param}")"
            ;;
            \#*) # line start with #
                # commented line, ignore it
            ;;
            *) # invalid format
                echo "\"$line\" is invalid configuration parameter format"
                echo "Use any of the following format\n-c 300\nor\n--conn-limit=300\nor\nconn-limit = 300"
                exit 1
            ;;
        esac
    done <"$configFile"
    cmd="$(echo "${cmd}" "${args}")"
}

# if configFile exist then parse it.
if [ -f "${configFile}" ]; then
    parseConfig
fi
# Now run docker-entrypoint.sh and send the parsed configs as arguments to it
$cmd
`
}
