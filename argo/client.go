package argo

import (
	"context"
	"fmt"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"path/filepath"

	"github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned"
	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type WorkflowClient struct {
	client    versioned.Interface
	namespace string
}

func NewWorkflowClient(namespace string) (*WorkflowClient, error) {
	// 尝试从默认位置加载 kubeconfig
	home := homedir.HomeDir()
	kubeconfig := filepath.Join(home, ".kube", "config")

	// 构建配置
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to build kubeconfig: %w", err)
	}

	// 创建 Argo Workflows 客户端
	client, err := versioned.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Argo client: %w", err)
	}

	return &WorkflowClient{
		client:    client,
		namespace: namespace,
	}, nil
}

func (c *WorkflowClient) SubmitWorkflow(workflow *v1alpha1.Workflow) (*v1alpha1.Workflow, error) {
	return c.client.ArgoprojV1alpha1().Workflows(c.namespace).Create(context.Background(), workflow, metav1.CreateOptions{})
}

func (c *WorkflowClient) GetWorkflow(name string) (*v1alpha1.Workflow, error) {
	return c.client.ArgoprojV1alpha1().Workflows(c.namespace).Get(context.Background(), name, metav1.GetOptions{})
}

func (c *WorkflowClient) ListWorkflows() (*v1alpha1.WorkflowList, error) {
	return c.client.ArgoprojV1alpha1().Workflows(c.namespace).List(context.Background(), metav1.ListOptions{})
}

func (c *WorkflowClient) DeleteWorkflow(name string) error {
	return c.client.ArgoprojV1alpha1().Workflows(c.namespace).Delete(context.Background(), name, metav1.DeleteOptions{})
}
