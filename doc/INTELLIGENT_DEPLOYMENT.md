# 智能发布系统实现文档

## 概述

本系统在原有灰度发布功能基础上，新增了以下核心能力：
1. **自动健康检测** - 实时监控发布节点的健康状态
2. **自动回滚** - 当检测到服务异常时自动触发回滚
3. **环境抽象层** - 统一的部署接口支持 K8S 和物理机环境

## 系统架构

```
┌─────────────────────────────────────────────────────────────────┐
│                      Deploy Manager (统一调度层)                  │
│  - 管理多种执行器                                                  │
│  - 协调健康监控和自动回滚                                          │
└────────┬───────────────────────────────────┬────────────────────┘
         │                                   │
         ▼                                   ▼
┌────────────────────┐              ┌────────────────────┐
│  Health Monitor    │              │ Auto Rollback      │
│  - 周期性健康检查   │              │ - 自动回滚执行      │
│  - 异常计数统计     │──触发──────▶│ - 状态更新         │
│  - 阈值判断        │              │ - 历史记录         │
└────────────────────┘              └────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────────────────────┐
│              Deployment Executor (执行器抽象层)                    │
├───────────────────────┬─────────────────────────────────────────┤
│   K8S Executor        │   Physical Machine Executor             │
│  - K8S API 调用        │  - SSH 远程执行                          │
│  - Pod 管理           │  - 包下载和部署                           │
│  - 健康检查           │  - 进程管理                              │
└───────────────────────┴─────────────────────────────────────────┘
```

## 核心组件

### 1. Deployment Executor (部署执行器)

#### 接口定义
```go
type DeploymentExecutor interface {
    Deploy(ctx context.Context, req *DeployRequest) error
    Rollback(ctx context.Context, req *RollbackRequest) error
    GetHealth(ctx context.Context, req *HealthCheckRequest) (*HealthStatus, error)
}
```

#### 实现类型

**K8S Executor** (`k8s_executor.go`)
- 支持 Kubernetes 环境部署
- 通过 K8S API 管理 Pod 生命周期
- 支持滚动更新和金丝雀发布
- 配置示例：
```go
K8SConfig{
    KubeConfigPath: "/path/to/kubeconfig",
    Namespace: "production",
    Timeout: 5 * time.Minute,
}
```

**Physical Executor** (`physical_executor.go`)
- 支持物理机/虚拟机环境部署
- 通过 SSH 执行远程命令
- 支持包下载、解压、启动等操作
- 配置示例：
```go
PhysicalConfig{
    SSHUser: "root",
    SSHKeyPath: "/path/to/ssh/key",
    Timeout: 5 * time.Minute,
}
```

### 2. Health Monitor (健康监控服务)

**功能特性：**
- 周期性检查灰度节点健康状态
- 支持自定义健康检查间隔和阈值
- 异常计数统计，避免偶发性故障误判
- 支持回调机制触发自动回滚

**配置参数：**
```go
MonitorConfig{
    CheckInterval: 30 * time.Second,  // 检查间隔
    HealthThreshold: 0.8,              // 健康率阈值 (80%)
}
```

**监控任务状态：**
```go
type MonitorTask struct {
    ReleaseID       string
    AppName         string
    DeviceType      string
    HealthURL       string      // 健康检查 URL
    NodeIDs         []string    // 监控的节点列表
    StartTime       time.Time   // 开始监控时间
    LastCheckTime   time.Time   // 最后检查时间
    LastHealthRate  float64     // 最后健康率
    CheckCount      int         // 检查次数
    UnhealthyCount  int         // 不健康次数
    Status          string      // 监控状态
}
```

**触发条件：**
- 健康率 < 阈值（默认 80%）
- 连续 3 次检查不通过
- 自动触发回滚流程

### 3. Auto Rollback Service (自动回滚服务)

**执行流程：**
1. 查询发布任务信息
2. 验证任务状态（必须是 `processing`）
3. 获取灰度节点列表
4. 执行回滚操作（恢复到 MainConfig 版本）
5. 清空灰度节点
6. 更新任务状态为 `rollbacked`
7. 记录操作历史

**回滚原因记录：**
```go
release.Describe = fmt.Sprintf("[AUTO-ROLLBACK] %s - %s", reason, release.Describe)
```

### 4. Deploy Manager (部署管理器)

统一管理和协调所有部署相关服务。

**主要功能：**
- 管理多个 Executor 实例
- 根据设备类型自动选择合适的执行器
- 启动和停止健康监控服务
- 连接健康监控和自动回滚服务
- 提供统一的部署和回滚接口

**使用示例：**
```go
deployManager, err := deploymanager.NewDeployManager(svcCtx)
if err != nil {
    return err
}

if err := deployManager.Start(ctx); err != nil {
    return err
}
defer deployManager.Stop()

if err := deployManager.Deploy(ctx, release, nodeIDs); err != nil {
    return err
}
```

## 数据模型

### 健康检查记录 (HealthCheckRecord)

```go
type HealthCheckRecord struct {
    ID             string    // 记录 ID
    ReleaseID      string    // 发布任务 ID
    AppName        string    // 应用名称
    DeviceType     string    // 设备类型
    HealthyNodes   []string  // 健康节点列表
    UnhealthyNodes []string  // 不健康节点列表
    TotalNodes     int       // 总节点数
    HealthyCount   int       // 健康节点数
    UnhealthyCount int       // 不健康节点数
    HealthRate     float64   // 健康率
    CheckTime      time.Time // 检查时间
    CreateAt       time.Time // 创建时间
}
```

MongoDB 集合：`healthCheck`

### 部署配置 (DeployConfig)

```go
type DeployConfig struct {
    ID           string       // 配置 ID
    DeviceType   string       // 设备类型
    ExecutorType ExecutorType // 执行器类型 (k8s/physical)
    K8SConfig    *K8SConfig   // K8S 配置
    PhysicalCfg  *PhysicalCfg // 物理机配置
    AutoRollback bool         // 是否启用自动回滚
    HealthCheck  *HealthCfg   // 健康检查配置
    CreateAt     time.Time    // 创建时间
    UpdateAt     time.Time    // 更新时间
}
```

MongoDB 集合：`deployConfig`

## 使用场景

### 场景 1：K8S 环境灰度发布

1. **配置部署环境**
```json
{
  "deviceType": "k8s-node",
  "executorType": "k8s",
  "k8sConfig": {
    "kubeConfigPath": "/etc/kubernetes/admin.conf",
    "namespace": "production",
    "timeoutSeconds": 300
  },
  "autoRollback": true,
  "healthCheck": {
    "checkInterval": 30,
    "healthThreshold": 0.8,
    "maxRetries": 3
  }
}
```

2. **创建发布任务**
```http
POST /v1/release/create
{
  "devType": "k8s-node",
  "appName": "example-service",
  "describe": "升级到 v2.0.0",
  "releaseType": "formal",
  "opType": "update",
  "appConfig": {
    "url": "https://kodo.example.com/packages/v2.0.0.tar.gz",
    "type": "tar.gz",
    "cmd": "example-service",
    "healthUrl": "/health",
    "md5": "..."
  },
  "grayPolicy": {
    "percentage": 10,
    "filter": {
      "status": "online",
      "stages": ["prod"]
    }
  }
}
```

3. **自动监控**
- 系统自动对灰度节点进行健康检查
- 每 30 秒检查一次健康状态
- 如果健康率低于 80% 且连续 3 次失败，触发自动回滚

4. **继续发布或自动回滚**
- 如果健康检查通过，可以继续增加灰度比例
- 如果检测到异常，系统自动回滚到 v1.0.0

### 场景 2：物理机环境发布

1. **配置部署环境**
```json
{
  "deviceType": "node",
  "executorType": "physical",
  "physicalConfig": {
    "sshUser": "deploy",
    "sshKeyPath": "/home/deploy/.ssh/id_rsa",
    "timeoutSeconds": 300
  },
  "autoRollback": true,
  "healthCheck": {
    "checkInterval": 60,
    "healthThreshold": 0.9,
    "maxRetries": 3
  }
}
```

2. **发布流程**
与 K8S 环境类似，但执行器会通过 SSH 连接到物理机执行部署操作

### 场景 3：多版本并行发布

```
时间轴：
├─ T1: 创建 v1.1 正式发布任务 (灰度 20%)
├─ T2: 创建 v1.2 功能验证任务 (灰度 10%)
├─ T3: v1.1 健康检查通过
├─ T4: v1.1 继续发布 (灰度 50%)
├─ T5: v1.2 健康检查失败，自动回滚
├─ T6: v1.1 全量发布完成
└─ T7: 修复 v1.2 问题后重新发布
```

## 监控与告警

### 健康检查日志

```
[HealthMonitor] Release 507f1f77 health check: 90/100 nodes healthy (90.00%)
[HealthMonitor] Release 507f1f77 health rate (75.00%) below threshold (80.00%), unhealthy count: 1
[HealthMonitor] Release 507f1f77 has failed health check 3 times, triggering auto-rollback
[AutoRollback] Starting auto-rollback for release 507f1f77, reason: Health check failed: 75.00% healthy nodes
[AutoRollback] Successfully completed auto-rollback for release 507f1f77
```

### 发布历史记录

自动回滚会在发布历史中记录：
```json
{
  "releaseId": "507f1f77bcf86cd799439011",
  "operation": "auto_rollback",
  "operator": "system",
  "opTime": 1698000000,
  "beforeState": "processing",
  "afterState": "rollbacked"
}
```

## 配置建议

### 健康检查阈值设置

| 环境类型 | 建议阈值 | 检查间隔 | 说明 |
|---------|---------|---------|------|
| 生产环境 | 90% | 30s | 高可用要求，快速发现问题 |
| 预发环境 | 80% | 60s | 允许一定容错，减少误报 |
| 测试环境 | 70% | 120s | 更宽松的阈值，减少干扰 |

### 回滚策略

| 失败次数 | 操作 | 说明 |
|---------|------|------|
| 1 次 | 警告日志 | 可能是偶发性问题 |
| 2 次 | 告警通知 | 需要关注但不立即回滚 |
| 3 次 | 自动回滚 | 确认问题持续存在 |

## 扩展性

### 添加新的执行器类型

1. 实现 `DeploymentExecutor` 接口
2. 注册到 `NewExecutor` 工厂函数
3. 添加相应的配置结构

示例：添加 Docker Swarm 支持
```go
type SwarmExecutor struct {
    config *SwarmConfig
}

func (e *SwarmExecutor) Deploy(ctx context.Context, req *DeployRequest) error {
    // 实现 Docker Swarm 部署逻辑
}

// 在 interface.go 中注册
case ExecutorTypeSwarm:
    return NewSwarmExecutor(config)
```

### 自定义健康检查逻辑

可以扩展健康检查方式：
- HTTP 健康检查（已实现）
- TCP 端口检查
- 自定义脚本检查
- 业务指标监控（QPS、错误率等）

## 安全考虑

1. **权限控制**
   - K8S 执行器需要适当的 RBAC 权限
   - SSH 执行器使用密钥认证，不支持密码

2. **网络安全**
   - 健康检查支持 HTTPS
   - SSH 连接加密传输

3. **审计日志**
   - 所有自动回滚操作记录在 `nodeReleaseHistory` 中
   - 操作人标记为 "system"

## 故障排查

### 自动回滚未触发

检查项：
1. `DeployConfig.AutoRollback` 是否设置为 `true`
2. 健康检查 URL 是否配置正确
3. 查看健康监控日志，确认是否达到触发条件

### 健康检查失败

可能原因：
1. 健康检查 URL 不可访问
2. 节点网络问题
3. 服务启动时间过长

### 部署失败

排查步骤：
1. 查看执行器日志
2. 验证包 URL 是否可访问
3. 检查目标节点权限和资源

## 未来优化方向

1. **智能阈值调整**
   - 基于历史数据自动调整健康阈值
   - 机器学习预测发布风险

2. **分级回滚策略**
   - 先回滚部分节点
   - 根据效果决定是否全量回滚

3. **多维度健康检查**
   - CPU、内存使用率
   - 业务指标（订单量、错误率）
   - 依赖服务健康状态

4. **可视化监控大盘**
   - 实时健康率曲线
   - 发布进度可视化
   - 告警通知集成

## 总结

本系统通过引入**执行器抽象层**、**健康监控服务**和**自动回滚机制**，实现了：

✅ **统一的部署接口** - 一套代码同时支持 K8S 和物理机环境  
✅ **自动故障检测** - 实时监控服务健康，及时发现问题  
✅ **智能自动回滚** - 异常时自动恢复，减少人工干预  
✅ **灵活可配置** - 支持不同环境的定制化配置  
✅ **良好扩展性** - 易于添加新的执行器类型和健康检查方式  

系统设计遵循**开闭原则**，通过接口抽象实现了对不同部署环境的支持，为后续功能扩展奠定了良好基础。
