package eventhooks

import (
	"bytes"
	"io"
	"strings"

	confiv1 "github.com/configurator/multitenancy/pkg/apis/confi/v1"
	"github.com/configurator/multitenancy/pkg/eventhooks/runner"
	"github.com/configurator/multitenancy/pkg/eventhooks/slack"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var clientset *kubernetes.Clientset
var log = logf.Log.WithName("event_hooks")

// RegisterConfig builds a package-local clientset for the managers rest.Config.
// This exposes functionality that the controller-runtime client has not implemented yet.
func RegisterConfig(config *rest.Config) error {
	var err error
	clientset, err = kubernetes.NewForConfig(config)
	return err
}

// ExecuteHook directs the hook to the correct function depending on the lifecycle event
// being triggered.
func ExecuteHook(ev confiv1.LifecycleEvent, hook confiv1.EventHook, mt *confiv1.MultiTenancy, tenant *confiv1.Tenant) error {
	runner := getRunner(hook)
	if runner == nil {
		log.Info("No configuration provided for event hook", "Hook", hook)
		return nil
	}
	switch ev {

	case confiv1.CreatedEvent:
		return runner.RunCreateHook(clientset, mt, tenant)

	case confiv1.UpdatedEvent:
		return runner.RunUpdateHook(clientset, mt, tenant)

	case confiv1.DeletedEvent:
		var logLines []string
		if hook.LogConfig != nil {
			var err error
			var skip bool
			if skip, logLines, err = getLogsForTenant(mt, tenant, hook.LogConfig); err != nil {
				return err
			}
			if skip {
				log.Info("Skipping hook fire due to unmatched logs")
				return nil
			}
		}
		return runner.RunDeleteHook(clientset, mt, tenant, logLines)
	}

	return nil
}

// getRunner returns the hook runner for the given eventhook configuration
func getRunner(hook confiv1.EventHook) runner.HookRunner {
	if hook.Slack != nil {
		return slack.NewRunner(hook.Slack)
	}
	return nil
}

// getLogsForTenant returns the requested logs for a tenant's pod. Skip is true
// if a provided regex is not matched in the logs.
func getLogsForTenant(mt *confiv1.MultiTenancy, tenant *confiv1.Tenant, cfg *confiv1.LogConfig) (skip bool, logLines []string, err error) {
	req := clientset.CoreV1().
		Pods(tenant.GetNamespace()).
		GetLogs(mt.GetCombinedName(tenant), &corev1.PodLogOptions{
			Container: cfg.GetContainer(),
			TailLines: cfg.GetTailLines(),
		})
	podLogs, err := req.Stream()
	if err != nil {
		return true, nil, err
	}
	defer podLogs.Close()
	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, podLogs)
	if err != nil {
		return true, nil, err
	}
	logs := buf.String()
	if regex := cfg.GetRegex(); regex != nil {
		if len(regex.FindAllString(logs, -1)) == 0 {
			return true, nil, nil
		}
	}
	return false, strings.Split(logs, "\n"), nil
}
