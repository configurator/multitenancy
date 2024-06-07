package slack

import (
	"fmt"

	confiv1 "github.com/configurator/multitenancy/pkg/apis/confi/v1"
	"github.com/configurator/multitenancy/pkg/eventhooks/runner"
	"k8s.io/client-go/kubernetes"
)

type slackRunner struct {
	config *confiv1.SlackConfig
}

func NewRunner(conf *confiv1.SlackConfig) runner.HookRunner {
	return &slackRunner{config: conf}
}

func (s *slackRunner) RunCreateHook(clientset *kubernetes.Clientset, mt *confiv1.MultiTenancy, tenant *confiv1.Tenant) error {
	return nil
}

func (s *slackRunner) RunUpdateHook(clientset *kubernetes.Clientset, mt *confiv1.MultiTenancy, tenant *confiv1.Tenant) error {
	return nil
}

func (s *slackRunner) RunDeleteHook(clientset *kubernetes.Clientset, mt *confiv1.MultiTenancy, tenant *confiv1.Tenant, logLines []string) error {
	fmt.Println("Received delete hook for tenant with logs", logLines)
	return nil
}
