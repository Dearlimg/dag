package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"dag/argo"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"gopkg.in/yaml.v2"
)

func main() {
	// 创建 Argo Workflow 客户端
	client, err := argo.NewWorkflowClient("argo")
	if err != nil {
		fmt.Printf("Failed to create Argo client: %v\n", err)
		fmt.Println("Note: This error is expected if you don't have a Kubernetes cluster configured")
		os.Exit(1)
	}

	// 读取工作流定义文件
	yamlFile, err := ioutil.ReadFile("basic-dag-workflow.yaml")
	if err != nil {
		fmt.Printf("Failed to read workflow file: %v\n", err)
		os.Exit(1)
	}

	// 解析工作流定义
	var workflow v1alpha1.Workflow
	if err := yaml.Unmarshal(yamlFile, &workflow); err != nil {
		fmt.Printf("Failed to unmarshal workflow: %v\n", err)
		os.Exit(1)
	}

	// 提交工作流
	submittedWorkflow, err := client.SubmitWorkflow(&workflow)
	if err != nil {
		fmt.Printf("Failed to submit workflow: %v\n", err)
		fmt.Println("Note: This error is expected if you don't have Argo Workflows installed")
		os.Exit(1)
	}

	fmt.Printf("Workflow submitted successfully: %s\n", submittedWorkflow.Name)

	// 列出所有工作流
	workflows, err := client.ListWorkflows()
	if err != nil {
		fmt.Printf("Failed to list workflows: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\nCurrent workflows:")
	for _, wf := range workflows.Items {
		fmt.Printf("- %s (Status: %s)\n", wf.Name, wf.Status.Phase)
	}
}
