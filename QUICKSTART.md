# 智能发布系统 - 快速开始指南

本指南帮助您在 5 分钟内快速上手智能发布系统。

## 📦 第一步：编译系统

### 前置要求
- Go 1.21+
- Make

### 编译命令

```bash
# 1. 克隆代码（如果还没有）
git clone https://github.com/PonnyS/Hackathon.git
cd Hackathon

# 2. 安装依赖
make deps

# 3. 编译程序
make build
```

✅ 编译成功后，您会看到：`bin/deployer`

## 🚀 第二步：运行示例

### 选项 A：运行主程序（演示所有功能）

```bash
make run
```

或

```bash
./bin/deployer
```

**您将看到**：
- K8S 环境的灰度发布演示
- 物理机环境的部署演示
- 多版本并行部署演示

### 选项 B：运行单个示例

```bash
# 编译所有示例
make build-all

# K8S 灰度发布示例
./bin/k8s_deployment

# 物理机部署示例
./bin/physical_deployment

# 并行部署示例
./bin/parallel_deployment
```

## 🐳 第三步：Docker 部署（可选）

```bash
# 构建镜像
make docker-build

# 运行容器
make docker-run
```

## ☸️ 第四步：Kubernetes 部署（可选）

### 前置要求
- Kubernetes 集群
- kubectl 命令

### 部署命令

```bash
# 一键部署
make k8s-deploy

# 查看状态
kubectl get pods -l app=intelligent-deployment-system

# 查看日志
kubectl logs -l app=intelligent-deployment-system -f
```

## 📖 示例输出

当您运行主程序时，您会看到类似的输出：

```
=== Intelligent Deployment System ===

Example 1: Deploy to Kubernetes with canary strategy
-------------------------------------------------------
[Orchestrator] Starting deployment of user-service:v1.2.0 (previous: v1.1.0)
[K8S] Deploying user-service version v1.2.0 with 5 replicas
[Orchestrator] Initial canary traffic: 10% to v1.2.0
[Rollback] Health check passed for user-service:v1.2.0
[Orchestrator] Canary progress: 20% traffic to v1.2.0 (overall: 20%)
[K8S] Setting traffic weights for user-service: map[v1.1.0:80 v1.2.0:20]
[Orchestrator] Canary progress: 30% traffic to v1.2.0 (overall: 30%)
...
[Orchestrator] Canary rollout completed successfully for user-service:v1.2.0
[Orchestrator] Successfully deployed user-service:v1.2.0

Example 2: Deploy to physical machines
---------------------------------------
[Orchestrator] Starting deployment of api-gateway:v2.0.0 (previous: v1.9.0)
[Physical] Deploying api-gateway version v2.0.0 to 3 instances
[Orchestrator] Initial canary traffic: 5% to v2.0.0
...

=== Deployment System Demo Completed ===
```

## 🎯 核心功能演示

### 1. 灰度发布
系统会逐步将流量从旧版本切换到新版本：
- 初始 10% 流量
- 每 30 秒增加 20%
- 持续健康检查
- 最终 100% 流量

### 2. 自动回滚
如果健康检查失败，系统会自动回滚：
```
[Orchestrator] Health check failed, initiating rollback
[Rollback] Rolling back user-service from v1.2.0 to v1.1.0
[Rollback] Successfully rolled back
```

### 3. 并行部署
可以同时部署多个版本：
```
Active deployments for user-service: 2
  - Version v1.2.0: deploying (progress: 40%)
  - Version v1.3.0: deploying (progress: 10%)
```

## 📝 配置自定义部署

### 编辑配置文件

配置文件位置：`config/deployment.yaml`

```yaml
services:
  your-service:
    environment: kubernetes
    health_check:
      type: http
      endpoint: /health
      timeout: 5s
    canary_strategy:
      initial_traffic: 10
      increment: 20
      interval: 1m
    replicas: 3
```

### 使用自定义配置

```bash
export DEPLOYMENT_CONFIG=/path/to/your/config.yaml
./bin/deployer
```

## 📚 更多文档

- **完整使用文档**: [README.md](README.md)
- **部署运维指南**: [docs/DEPLOYMENT_GUIDE.md](docs/DEPLOYMENT_GUIDE.md)
- **设计需求文档**: [docs/DESIGN_REQUIREMENTS.md](docs/DESIGN_REQUIREMENTS.md)

## 🆘 遇到问题？

### 编译失败

```bash
go clean -modcache
make deps
make build
```

### 运行失败

检查 Go 版本：
```bash
go version  # 应该 >= 1.21
```

### 需要帮助

- 查看完整文档：[DEPLOYMENT_GUIDE.md](docs/DEPLOYMENT_GUIDE.md)
- 提交 Issue：https://github.com/PonnyS/Hackathon/issues

## ✅ 完成！

现在您已经：
- ✅ 编译了系统
- ✅ 运行了示例
- ✅ 了解了核心功能

下一步：查看 [DEPLOYMENT_GUIDE.md](docs/DEPLOYMENT_GUIDE.md) 了解生产环境部署。

---

**预计用时**: 5 分钟  
**难度**: 简单 ⭐
