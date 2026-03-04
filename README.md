# DAG 流水线系统

## 项目简介

本项目展示了从简陋的本地 DAG 实现到使用 ArgoFlow 重构的完整过程。包含两个阶段的实现：

1. **本地 DAG 实现**：基于 Go 语言的简单 DAG 流水线系统
2. **ArgoFlow 迁移方案**：使用 Kubernetes 生态中的 Argo Workflows 进行重构的完整设计方案

## 目录结构

```
dag/
├── internal/             # 内部包
│   └── dag/            # 本地 DAG 实现
│       └── dag.go      # 核心 DAG 实现代码
├── design/              # 设计文档
│   └── argoflow.md     # ArgoFlow 设计文档
├── ARGOFLOW_MIGRATION.md  # ArgoFlow 迁移方案
├── main.go             # 示例程序
├── go.mod              # Go 模块文件
└── README.md           # 本说明文件
```

## 本地 DAG 实现

### 功能特性

- **任务依赖管理**：支持复杂的任务依赖关系
- **并发执行**：自动并行执行无依赖的任务
- **错误处理**：支持失败即停止和继续执行两种模式
- **重试机制**：支持任务失败自动重试
- **执行状态追踪**：提供详细的执行状态和统计信息

### 快速开始

1. **运行示例**

```bash
# 运行示例程序
go run main.go
```

2. **示例说明**

示例程序包含三个演示场景：
- **示例 1**：基础流水线执行
- **示例 2**：带重试机制的流水线
- **示例 3**：错误处理（继续执行模式）

### 核心 API

```go
// 创建新的流水线
pipeline := dag.NewPipeline()

// 添加任务
task1 := dag.NewTask("task1", "数据准备", func() error {
    // 任务逻辑
    return nil
})

// 添加依赖任务
task2 := dag.NewTask("task2", "数据处理", func() error {
    // 任务逻辑
    return nil
}, "task1")

// 执行流水线
if err := pipeline.Execute(); err != nil {
    fmt.Printf("流水线执行失败: %v\n", err)
}

// 获取执行摘要
summary := pipeline.GetExecutionSummary()
```

## ArgoFlow 迁移方案

### 迁移原因

- **高可用性**：基于 Kubernetes 的分布式架构，提供更好的可靠性
- **企业级特性**：内置监控、告警、权限管理等企业级功能
- **可扩展性**：支持水平扩展，处理大规模工作流
- **生态集成**：与 Kubernetes 生态系统深度集成
- **降低维护成本**：由社区维护，持续更新

### 详细迁移方案

请参考 [ARGOFLOW_MIGRATION.md](design/ARGOFLOW_MIGRATION.md) 文件，包含完整的迁移设计和实施计划。

## 技术栈对比

| 特性 | 本地 DAG 实现 | ArgoFlow |
|------|--------------|----------|
| 部署方式 | 单机进程 | Kubernetes 集群 |
| 高可用性 | 低 | 高 |
| 扩展性 | 有限 | 无限 |
| 维护成本 | 高 | 低 |
| 集成能力 | 有限 | 强 |
| 企业级特性 | 基础 | 丰富 |

## 相关文档

- [ARGOFLOW_MIGRATION.md](design/ARGOFLOW_MIGRATION.md) - ArgoFlow 迁移完整方案
- [设计文档](design/argoflow.md) - ArgoFlow 设计细节

## 贡献

欢迎提交问题和改进建议！

## 许可证

MIT License
