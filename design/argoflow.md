# ArgoFlow 替换现有 DAG 系统设计方案

## 1. 现状分析

### 现有 DAG 系统功能
- 任务依赖管理
- 并发执行
- 错误处理
- 重试机制
- 执行状态追踪

### 存在问题
- 单点故障风险
- 无高可用保障
- 缺乏企业级特性
- 维护成本高

## 2. ArgoFlow 方案设计

### 2.1 架构设计

```
┌───────────────────────────────────────────────────────────────┐
│                       ArgoFlow 架构                          │
├───────────────────────────────────────────────────────────────┤
│  ┌────────────┐  ┌──────────────┐  ┌───────────────────────┐ │
│  │  客户端 API  │→ │  Argo Server  │→ │  Argo Workflow CRD  │ │
│  └────────────┘  └──────────────┘  └───────────────────────┘ │
│                    │                 │                       │
│                    ▼                 ▼                       │
│  ┌────────────────────────────────────────────────────────┐  │
│  │                   Kubernetes 集群                     │  │
│  │  ┌────────────┐  ┌────────────┐  ┌────────────┐       │  │
│  │  │  Worker 1  │  │  Worker 2  │  │  Worker N  │       │  │
│  │  └────────────┘  └────────────┘  └────────────┘       │  │
│  └────────────────────────────────────────────────────────┘  │
└───────────────────────────────────────────────────────────────┘
```

### 2.2 核心组件

1. **Argo Server**：提供 REST API 和 UI 界面
2. **Argo Workflow CRD**：自定义资源定义，管理工作流
3. **Kubernetes 集群**：提供容器编排和资源管理
4. **Worker 节点**：执行具体任务

## 3. 具体实现细节

### 3.1 安装配置

#### 3.1.1 安装 Argo Workflows

```bash
# 创建命名空间
kubectl create namespace argo

# 安装 Argo Workflows
kubectl apply -n argo -f https://github.com/argoproj/argo-workflows/releases/download/v3.5.4/install.yaml

# 配置 Argo Server 服务
kubectl patch configmap workflow-controller-configmap -n argo --type merge -p '{"data": {"artifactRepository": "{\"s3\": {\"bucket\": \"my-bucket\", \"endpoint\": \"minio:9000\", \"insecure\": true, \"accessKeySecret\": {\"name\": \"minio-cred\", \"key\": \"accesskey\"}, \"secretKeySecret\": {\"name\": \"minio-cred\", \"key\": \"secretkey\"}}}"}}'
```

#### 3.1.2 配置认证和授权

```bash
# 创建 ServiceAccount
kubectl create serviceaccount argo-workflow -n argo

# 绑定权限
kubectl create rolebinding argo-workflow-rolebinding -n argo --clusterrole=edit --serviceaccount=argo:argo-workflow
```

### 3.2 工作流定义

#### 3.2.1 基础 DAG 工作流

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: data-processing-workflow
  namespace: argo
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: prepare-data
        template: prepare-data
      - name: process-data-a
        template: process-data
        dependencies: [prepare-data]
        arguments:
          parameters:
          - name: type
            value: "A"
      - name: process-data-b
        template: process-data
        dependencies: [prepare-data]
        arguments:
          parameters:
          - name: type
            value: "B"
      - name: aggregate-results
        template: aggregate-results
        dependencies: [process-data-a, process-data-b]

  - name: prepare-data
    container:
      image: your-registry/data-preparer:v1
      command: ["/app/prepare.sh"]

  - name: process-data
    inputs:
      parameters:
      - name: type
    container:
      image: your-registry/data-processor:v1
      command: ["/app/process.sh", "{{inputs.parameters.type}}"]

  - name: aggregate-results
    container:
      image: your-registry/aggregator:v1
      command: ["/app/aggregate.sh"]
```

#### 3.2.2 高级特性配置

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: advanced-workflow
  namespace: argo
spec:
  entrypoint: main
  retryStrategy:
    limit: 3
    backoff:
      duration: "1m"
      factor: 2
      maxDuration: "10m"
  templates:
  - name: main
    dag:
      tasks:
      - name: task-with-timeout
        template: task-with-timeout
        timeout: "30m"
      - name: task-with-retry
        template: task-with-retry
        retryStrategy:
          limit: 5
          retryPolicy: "Always"

  - name: task-with-timeout
    container:
      image: your-registry/task:v1
      command: ["/app/long-running-task.sh"]

  - name: task-with-retry
    container:
      image: your-registry/flakey-task:v1
      command: ["/app/flakey-task.sh"]
```

### 3.3 与现有系统集成

#### 3.3.1 API 集成

```go
package argo

import (
    "context"
    "fmt"
    "github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned"
    "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type WorkflowClient struct {
    client versioned.Interface
    namespace string
}

func NewWorkflowClient(kubeconfig string, namespace string) (*WorkflowClient, error) {
    // 初始化 Kubernetes 客户端
    config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
    if err != nil {
        return nil, err
    }
    
    client, err := versioned.NewForConfig(config)
    if err != nil {
        return nil, err
    }
    
    return &WorkflowClient{
        client: client,
        namespace: namespace,
    }, nil
}

func (c *WorkflowClient) SubmitWorkflow(workflow *v1alpha1.Workflow) (*v1alpha1.Workflow, error) {
    return c.client.ArgoprojV1alpha1().Workflows(c.namespace).Create(context.Background(), workflow, metav1.CreateOptions{})
}

func (c *WorkflowClient) GetWorkflow(name string) (*v1alpha1.Workflow, error) {
    return c.client.ArgoprojV1alpha1().Workflows(c.namespace).Get(context.Background(), name, metav1.GetOptions{})
}
```

#### 3.3.2 事件触发集成

```yaml
apiVersion: argoproj.io/v1alpha1
kind: EventSource
metadata:
  name: github-events
  namespace: argo
spec:
  github:
    webhook:
      repository: your-org/your-repo
      events:
        - push
      webhookSecret:
        name: github-secret
        key: secret
      insecure: true

---
apiVersion: argoproj.io/v1alpha1
kind: Sensor
metadata:
  name: github-sensor
  namespace: argo
spec:
  dependencies:
    - name: github-event
      eventSourceName: github-events
      eventName: webhook
  triggers:
    - template:
        name: workflow-trigger
        k8s:
          operation: create
          source:
            resource:
              apiVersion: argoproj.io/v1alpha1
              kind: Workflow
              metadata:
                generateName: github-triggered-
              spec:
                entrypoint: main
                templates:
                  - name: main
                    container:
                      image: your-registry/build-task:v1
                      command: ["/app/build.sh"]
```

### 3.4 监控与维护

#### 3.4.1 Prometheus 监控

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: argo-workflows
  namespace: monitoring
spec:
  selector:
    matchLabels:
      app: workflow-controller
  endpoints:
  - port: metrics
    interval: 30s
```

#### 3.4.2 Grafana 仪表板

导入 Argo 官方提供的 Grafana 仪表板：
- 仪表板 ID: 11525 (Argo Workflows)

### 3.5 存储配置

#### 3.5.1 配置 MinIO 作为制品存储

```bash
# 安装 MinIO
helm repo add bitnami https://charts.bitnami.com/bitnami
helm install minio bitnami/minio -n argo --set auth.rootUser=admin --set auth.rootPassword=password

# 创建存储凭证
kubectl create secret generic minio-cred -n argo --from-literal=accesskey=admin --from-literal=secretkey=password
```

## 4. 迁移策略

### 4.1 渐进式迁移

1. **并行运行**：同时运行旧系统和 ArgoFlow
2. **功能验证**：验证 ArgoFlow 功能与旧系统一致性
3. **逐步切换**：优先迁移非关键任务
4. **完全迁移**：当验证通过后，完全切换到 ArgoFlow

### 4.2 数据迁移

1. **任务定义迁移**：将现有任务定义转换为 Argo Workflow YAML
2. **历史数据**：可选择性地将历史执行记录迁移到新系统
3. **配置迁移**：迁移相关配置和环境变量

## 5. 优势对比

| 特性 | 现有系统 | ArgoFlow |
|------|---------|----------|
| 高可用性 | 低 | 高（基于 K8s） |
| 扩展性 | 有限 | 无限（支持水平扩展） |
| 企业级特性 | 缺乏 | 丰富（监控、告警、权限管理） |
| 维护成本 | 高（需自行维护） | 低（社区支持） |
| 集成能力 | 有限 | 强（与 K8s 生态集成） |
| 可观测性 | 基础 | 高级（Prometheus、Grafana 集成） |

## 6. 实施计划

### 6.1 阶段一：环境准备（1-2 周）
- 安装 Kubernetes 集群
- 部署 Argo Workflows
- 配置存储和监控

### 6.2 阶段二：工作流定义（2-3 周）
- 转换现有任务为 Argo 工作流
- 测试工作流执行
- 优化配置

### 6.3 阶段三：集成与测试（2-3 周）
- 集成现有系统
- 性能测试
- 故障演练

### 6.4 阶段四：生产部署（1-2 周）
- 灰度发布
- 监控上线
- 回滚计划

## 7. 结论

使用 ArgoFlow 替换现有 DAG 系统是一个明智的选择，它提供了更强大、更可靠、更易于维护的工作流解决方案。通过本设计方案，可以平滑地完成从现有系统到 ArgoFlow 的迁移，同时获得企业级的功能和可靠性。
