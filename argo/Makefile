# Makefile for Argo Workflows integration

.PHONY: help build run submit-workflow list-workflows

help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  help              Show this help message"
	@echo "  build             Build the Argo client"
	@echo "  run               Run the Argo client example"
	@echo "  submit-workflow   Submit a workflow to Argo"
	@echo "  list-workflows    List all workflows in the argo namespace"

build:
	@echo "Building Argo client..."
	@go build -o argo-client main.go

run:
	@echo "Running Argo client example..."
	@go run main.go

submit-workflow:
	@echo "Submitting workflow to Argo..."
	@kubectl apply -f basic-dag-workflow.yaml

list-workflows:
	@echo "Listing workflows in argo namespace..."
	@kubectl get workflows -n argo
