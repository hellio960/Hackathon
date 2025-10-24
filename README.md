# 智能发布系统 (Intelligent Deployment System)

一个支持灰度发布、自动回滚、多版本并行部署的智能发布系统，统一支持 Kubernetes 和物理机环境。

## 核心特性

### ✅ 已实现的功能

- **灰度发布 (Canary Deployment)**: 支持按流量比例逐步发布新版本
- **自动健康检查**: 持续监控服务健康状态
- **自动回滚**: 检测到问题时自动回滚到上一个稳定版本
- **多版本并行部署**: 支持同时部署多个版本（如 v1.1 升级未完成时可以开始 v1.2）
- **环境抽象**: 统一的接口支持 K8S 和物理机环境
- **可配置策略**: 灵活配置发布策略和回滚策略

## 系统架构

```
┌─────────────────────────────────────────────────────┐
│            Deployment Orchestrator                   │
│  (协调灰度发布流程、管理发布生命周期)                │
└────────────┬────────────────────────────┬───────────┘
             │                            │
    ┌────────▼────────┐          ┌───────▼────────┐
    │ Version Manager │          │ Health Checker │
    │  (版本管理)     │          │  (健康检查)    │
    └─────────────────┘          └────────┬───────┘
                                           │
                                  ┌────────▼─────────┐
                                  │Rollback Controller│
                                  │  (回滚控制)      │
                                  └────────┬─────────┘
                                           │
                         ┌─────────────────▼─────────────────┐
                         │    Environment Abstraction        │
                         │      (环境抽象层)                 │
                         └─────────┬──────────────┬──────────┘
                                   │              │
                        ┌──────────▼─────┐  ┌────▼──────────┐
                        │   Kubernetes   │  │   Physical    │
                        │   Environment  │  │  Environment  │
                        └────────────────┘  └───────────────┘
```

## 核心组件

### 1. 环境抽象层 (Environment Abstraction)
- 统一 K8S 和物理机的部署接口
- 支持流量控制和实例管理
- 可扩展支持更多环境类型

### 2. 版本管理器 (Version Manager)
- 管理多个版本的并行部署
- 跟踪每个版本的部署状态和进度
- 支持版本间的状态转换

### 3. 健康检查器 (Health Checker)
- 支持 HTTP、TCP 等多种健康检查方式
- 可配置的健康阈值
- 实时监控服务实例健康状态

### 4. 回滚控制器 (Rollback Controller)
- 自动检测服务问题
- 支持多种回滚触发条件
- 记录回滚历史

### 5. 部署编排器 (Orchestrator)
- 协调整个灰度发布流程
- 控制流量切换策略
- 管理部署生命周期

## 快速开始

### 安装依赖

```bash
go mod download
```

### 运行示例

#### 1. 基础示例 - 演示所有功能

```bash
go run cmd/deployer/main.go
```

#### 2. Kubernetes 环境部署

```bash
go run examples/k8s_deployment.go
```

#### 3. 物理机环境部署

```bash
go run examples/physical_deployment.go
```

#### 4. 多版本并行部署

```bash
go run examples/parallel_deployment.go
```

## 使用示例

### Kubernetes 环境灰度发布

```go
// 创建 K8S 环境
env, _ := environment.NewEnvironment("kubernetes", map[string]interface{}{
    "namespace": "production",
})

// 创建健康检查器和回滚控制器
healthChecker := health.NewHealthChecker(0.8)
rollbackCtrl := rollback.NewRollbackController(env, healthChecker, &rollback.RollbackPolicy{
    MaxHealthCheckFailures: 3,
    HealthCheckInterval:    30 * time.Second,
    AutoRollback:           true,
})

// 创建编排器
orch := orchestrator.NewOrchestrator(healthChecker, rollbackCtrl)

// 配置部署
config := &orchestrator.DeploymentConfig{
    Service:         "user-service",
    Version:         "v1.2.0",
    PreviousVersion: "v1.1.0",
    Environment:     env,
    Strategy: &orchestrator.CanaryStrategy{
        InitialTraffic: 10,    // 初始 10% 流量
        Increment:      10,    // 每次增加 10%
        Interval:       2 * time.Minute,
        MaxTraffic:     100,
    },
    HealthCheck: &health.HealthCheck{
        Type:     "http",
        Endpoint: "/health",
        Timeout:  5 * time.Second,
        Interval: 10 * time.Second,
    },
    Replicas: 5,
}

// 执行部署
err := orch.Deploy(context.Background(), config)
```

### 物理机环境部署

```go
// 创建物理机环境
env, _ := environment.NewEnvironment("physical", map[string]interface{}{
    "hosts": []string{"192.168.1.10", "192.168.1.11", "192.168.1.12"},
    "ssh": map[string]interface{}{
        "user":     "deploy",
        "key_path": "~/.ssh/deploy_key",
    },
})

// 其他配置类似...
```

## 配置文件

系统支持通过 YAML 配置文件管理部署策略：

```yaml
environments:
  kubernetes:
    type: kubernetes
    namespace: production
  
  physical:
    type: physical
    hosts:
      - 192.168.1.10
      - 192.168.1.11

services:
  user-service:
    environment: kubernetes
    health_check:
      type: http
      endpoint: /health
      timeout: 5s
    canary_strategy:
      initial_traffic: 10
      increment: 10
      interval: 2m
    rollback_policy:
      auto_rollback: true
```

## 关键特性说明

### 1. 灰度发布策略

- **初始流量 (InitialTraffic)**: 新版本初始接收的流量百分比
- **增量 (Increment)**: 每次增加的流量百分比
- **间隔 (Interval)**: 每次流量增加的时间间隔
- **最大流量 (MaxTraffic)**: 目标流量百分比（通常为 100%）

### 2. 健康检查

- 支持 HTTP、TCP 等多种检查方式
- 可配置健康阈值（默认 80%）
- 持续监控，实时反馈

### 3. 自动回滚

触发条件：
- 连续健康检查失败次数超过阈值
- 错误率超过配置阈值
- 手动触发

回滚过程：
- 自动切换流量到上一个稳定版本
- 记录回滚事件和原因
- 可查询回滚历史

### 4. 多版本并行

- 支持同一服务多个版本同时部署
- 独立的状态管理和进度跟踪
- 互不干扰的灰度策略执行

## 项目结构

```
.
├── cmd/
│   └── deployer/           # 主程序入口
├── pkg/
│   ├── environment/        # 环境抽象层
│   ├── version/           # 版本管理
│   ├── health/            # 健康检查
│   ├── rollback/          # 回滚控制
│   └── orchestrator/      # 部署编排
├── config/                # 配置文件
├── examples/              # 使用示例
└── README.md
```

## 扩展性

系统采用插件化设计，可轻松扩展：

1. **新增环境类型**: 实现 `Environment` 接口
2. **新增健康检查方式**: 扩展 `HealthCheck` 类型
3. **自定义回滚策略**: 配置 `RollbackPolicy`
4. **自定义灰度策略**: 实现 `CanaryStrategy`

## 后续优化方向

- [ ] 完善 K8S 客户端集成
- [ ] 实现物理机 SSH 部署
- [ ] 添加 Prometheus 指标采集
- [ ] 支持蓝绿部署策略
- [ ] Web UI 控制台
- [ ] 部署审批流程

## License

MIT
