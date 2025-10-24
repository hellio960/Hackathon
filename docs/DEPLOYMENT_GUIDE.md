# 智能发布系统 - 部署运维指南

## 目录

1. [环境要求](#环境要求)
2. [编译构建](#编译构建)
3. [本地运行](#本地运行)
4. [Docker 部署](#docker-部署)
5. [Kubernetes 部署](#kubernetes-部署)
6. [物理机部署](#物理机部署)
7. [配置说明](#配置说明)
8. [监控运维](#监控运维)
9. [故障排查](#故障排查)
10. [升级维护](#升级维护)

---

## 环境要求

### 开发环境

- **Go**: 1.21 或更高版本
- **Make**: 构建工具
- **Git**: 版本控制

### 运行环境

#### Kubernetes 环境
- Kubernetes 1.20+
- kubectl 命令行工具
- 集群管理员权限（用于创建 RBAC 资源）

#### 物理机环境
- Linux/Unix 操作系统
- SSH 访问权限
- Nginx 或 HAProxy（用于流量控制）

### 可选工具

- **Docker**: 容器化部署
- **Helm**: Kubernetes 包管理（未来版本）
- **Prometheus**: 监控指标采集
- **Grafana**: 监控可视化

---

## 编译构建

### 1. 克隆代码

```bash
git clone https://github.com/PonnyS/Hackathon.git
cd Hackathon
```

### 2. 安装依赖

```bash
make deps
```

这个命令会：
- 下载所有 Go 依赖
- 运行 `go mod tidy` 清理依赖

### 3. 编译主程序

```bash
make build
```

编译产物：`bin/deployer`

### 4. 编译所有程序（包括示例）

```bash
make build-all
```

编译产物：
- `bin/deployer` - 主程序
- `bin/k8s_deployment` - K8S 部署示例
- `bin/physical_deployment` - 物理机部署示例
- `bin/parallel_deployment` - 并行部署示例

### 5. 运行测试

```bash
make test
```

### 6. 清理编译产物

```bash
make clean
```

---

## 本地运行

### 快速开始

```bash
# 编译并运行主程序
make run

# 或直接运行
./bin/deployer
```

### 运行示例程序

#### 1. Kubernetes 部署示例

```bash
make run-k8s
```

或

```bash
./bin/k8s_deployment
```

**输出示例**：
```
[Orchestrator] Starting deployment of user-service:v1.2.0 (previous: v1.1.0)
[K8S] Deploying user-service version v1.2.0 with 5 replicas
[Orchestrator] Initial canary traffic: 10% to v1.2.0
[Rollback] Health check passed for user-service:v1.2.0
[Orchestrator] Canary progress: 20% traffic to v1.2.0 (overall: 20%)
...
[Orchestrator] Successfully deployed user-service:v1.2.0
```

#### 2. 物理机部署示例

```bash
make run-physical
```

#### 3. 并行部署示例

```bash
make run-parallel
```

### 配置文件

默认配置文件位置：`config/deployment.yaml`

可以通过环境变量指定：
```bash
export DEPLOYMENT_CONFIG=/path/to/your/config.yaml
./bin/deployer
```

---

## Docker 部署

### 1. 构建 Docker 镜像

```bash
make docker-build
```

或指定版本：

```bash
make docker-build VERSION=v1.0.0
```

这会创建镜像：`intelligent-deployment-system:v1.0.0`

### 2. 运行 Docker 容器

```bash
make docker-run
```

或使用 docker 命令：

```bash
docker run --rm -it \
  -v $(pwd)/config:/app/config \
  intelligent-deployment-system:latest
```

### 3. 挂载配置文件

```bash
docker run --rm -it \
  -v /path/to/config.yaml:/app/config/deployment.yaml \
  -e DEPLOYMENT_CONFIG=/app/config/deployment.yaml \
  intelligent-deployment-system:latest
```

### 4. 查看容器日志

```bash
docker logs -f <container-id>
```

---

## Kubernetes 部署

### 前置条件

1. 有可用的 Kubernetes 集群
2. kubectl 已配置并可访问集群
3. 有集群管理员权限

### 部署步骤

#### 1. 检查集群连接

```bash
kubectl cluster-info
kubectl get nodes
```

#### 2. 创建命名空间（可选）

```bash
kubectl create namespace deployment-system
```

如果使用自定义命名空间，需要修改 `deployments/kubernetes/deployment.yaml` 中的 namespace。

#### 3. 部署系统

```bash
make k8s-deploy
```

这个命令会创建：
- Deployment（3个副本）
- Service（ClusterIP）
- ConfigMap（配置文件）
- ServiceAccount（服务账号）
- ClusterRole 和 ClusterRoleBinding（RBAC 权限）
- HorizontalPodAutoscaler（自动扩缩容）

#### 4. 验证部署

```bash
# 查看 Pod 状态
kubectl get pods -l app=intelligent-deployment-system

# 查看 Service
kubectl get svc -l app=intelligent-deployment-system

# 查看详细信息
kubectl describe deployment intelligent-deployment-system
```

#### 5. 查看日志

```bash
# 查看所有 Pod 日志
kubectl logs -l app=intelligent-deployment-system --tail=100 -f

# 查看特定 Pod 日志
kubectl logs <pod-name> -f
```

#### 6. 访问服务

```bash
# 端口转发到本地
kubectl port-forward svc/intelligent-deployment-system 8080:80

# 访问
curl http://localhost:8080/health
```

### 高级配置

#### 启用 Ingress

创建 `deployments/kubernetes/ingress.yaml`:

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: intelligent-deployment-system
  annotations:
    kubernetes.io/ingress.class: nginx
spec:
  rules:
  - host: deployment.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: intelligent-deployment-system
            port:
              number: 80
```

应用：
```bash
kubectl apply -f deployments/kubernetes/ingress.yaml
```

#### 配置持久化存储

如需持久化数据（如日志、状态），添加 PersistentVolumeClaim：

```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: deployment-system-data
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 10Gi
```

#### 配置资源限制

修改 `deployment.yaml` 中的资源配置：

```yaml
resources:
  requests:
    memory: "512Mi"
    cpu: "500m"
  limits:
    memory: "1Gi"
    cpu: "1000m"
```

### 删除部署

```bash
make k8s-delete
```

或手动删除：

```bash
kubectl delete -f deployments/kubernetes/
```

---

## 物理机部署

### 部署架构

```
┌──────────────┐
│ Load Balancer│  (Nginx/HAProxy)
└──────┬───────┘
       │
   ┌───┴────┬────────┬────────┐
   │        │        │        │
┌──▼───┐ ┌──▼───┐ ┌──▼───┐ ┌──▼───┐
│Host 1│ │Host 2│ │Host 3│ │Host N│
│:8080 │ │:8080 │ │:8080 │ │:8080 │
└──────┘ └──────┘ └──────┘ └──────┘
```

### 部署步骤

#### 1. 准备服务器

```bash
# 在每台服务器上创建目录
sudo mkdir -p /opt/deployment-system
sudo chown $USER:$USER /opt/deployment-system
```

#### 2. 上传程序

```bash
# 编译
make build

# 上传到服务器
for host in host1 host2 host3; do
  scp bin/deployer $host:/opt/deployment-system/
  scp config/deployment.yaml $host:/opt/deployment-system/config/
done
```

#### 3. 创建 systemd 服务

在每台服务器上创建 `/etc/systemd/system/deployment-system.service`:

```ini
[Unit]
Description=Intelligent Deployment System
After=network.target

[Service]
Type=simple
User=deployuser
WorkingDirectory=/opt/deployment-system
ExecStart=/opt/deployment-system/deployer
Restart=always
RestartSec=10
Environment="DEPLOYMENT_CONFIG=/opt/deployment-system/config/deployment.yaml"

[Install]
WantedBy=multi-user.target
```

#### 4. 启动服务

```bash
# 在每台服务器上
sudo systemctl daemon-reload
sudo systemctl enable deployment-system
sudo systemctl start deployment-system

# 检查状态
sudo systemctl status deployment-system
```

#### 5. 配置负载均衡

**Nginx 配置示例** (`/etc/nginx/conf.d/deployment-system.conf`):

```nginx
upstream deployment_system {
    least_conn;
    server host1:8080 weight=1 max_fails=3 fail_timeout=30s;
    server host2:8080 weight=1 max_fails=3 fail_timeout=30s;
    server host3:8080 weight=1 max_fails=3 fail_timeout=30s;
}

server {
    listen 80;
    server_name deployment.example.com;

    location / {
        proxy_pass http://deployment_system;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_connect_timeout 30s;
        proxy_send_timeout 30s;
        proxy_read_timeout 30s;
    }

    location /health {
        proxy_pass http://deployment_system/health;
        access_log off;
    }
}
```

重载 Nginx:
```bash
sudo nginx -t
sudo systemctl reload nginx
```

### 日志管理

#### 查看日志

```bash
# systemd 日志
sudo journalctl -u deployment-system -f

# 应用日志（如果配置了文件日志）
tail -f /var/log/deployment-system/app.log
```

#### 配置日志轮转

创建 `/etc/logrotate.d/deployment-system`:

```
/var/log/deployment-system/*.log {
    daily
    rotate 30
    compress
    delaycompress
    notifempty
    create 0640 deployuser deployuser
    sharedscripts
    postrotate
        systemctl reload deployment-system > /dev/null 2>&1 || true
    endscript
}
```

---

## 配置说明

### 配置文件结构

配置文件：`config/deployment.yaml`

```yaml
# 环境配置
environments:
  # Kubernetes 环境
  kubernetes:
    type: kubernetes
    namespace: production
    kubeconfig: ~/.kube/config  # 可选，默认使用 in-cluster config
  
  # 物理机环境
  physical:
    type: physical
    hosts:
      - 192.168.1.10
      - 192.168.1.11
      - 192.168.1.12
    ssh:
      user: deploy
      key_path: ~/.ssh/deploy_key
      port: 22

# 服务配置
services:
  # 服务名称
  user-service:
    # 使用的环境
    environment: kubernetes
    
    # 健康检查配置
    health_check:
      type: http              # http, tcp, script
      endpoint: /health       # 健康检查端点
      timeout: 5s            # 超时时间
      interval: 10s          # 检查间隔
      expected_status: 200   # 期望的 HTTP 状态码（仅 http）
    
    # 灰度策略
    canary_strategy:
      initial_traffic: 10    # 初始流量比例 (%)
      increment: 10         # 每次增加的流量 (%)
      interval: 2m          # 每次增量的间隔
      max_traffic: 100      # 最大流量比例 (%)
    
    # 副本数
    replicas: 5
    
    # 回滚策略
    rollback_policy:
      max_health_check_failures: 3  # 最大失败次数
      health_check_interval: 30s    # 检查间隔
      auto_rollback: true          # 是否自动回滚
      error_rate_threshold: 0.1    # 错误率阈值 (10%)
```

### 环境变量

系统支持以下环境变量：

| 变量名 | 说明 | 默认值 |
|--------|------|--------|
| `DEPLOYMENT_CONFIG` | 配置文件路径 | `config/deployment.yaml` |
| `APP_ENV` | 运行环境 | `development` |
| `LOG_LEVEL` | 日志级别 | `info` |
| `HTTP_PORT` | HTTP 服务端口 | `8080` |

示例：
```bash
export DEPLOYMENT_CONFIG=/etc/deployment/config.yaml
export APP_ENV=production
export LOG_LEVEL=debug
./bin/deployer
```

---

## 监控运维

### 健康检查

系统提供以下健康检查端点：

```bash
# 存活检查
curl http://localhost:8080/health

# 就绪检查
curl http://localhost:8080/ready

# 详细状态
curl http://localhost:8080/status
```

### 监控指标

系统暴露 Prometheus 格式的监控指标（如果启用）：

```bash
curl http://localhost:8080/metrics
```

关键指标：
- `deployment_total` - 部署总数
- `deployment_success_total` - 成功部署数
- `deployment_failed_total` - 失败部署数
- `rollback_total` - 回滚总数
- `deployment_duration_seconds` - 部署耗时
- `health_check_failures_total` - 健康检查失败次数

### 日志级别

支持的日志级别：
- `debug` - 调试信息
- `info` - 一般信息（默认）
- `warn` - 警告信息
- `error` - 错误信息

动态调整日志级别：
```bash
curl -X POST http://localhost:8080/admin/log-level -d '{"level":"debug"}'
```

---

## 故障排查

### 常见问题

#### 1. 编译失败

**问题**：`go build` 失败

**解决**：
```bash
# 清理依赖
go clean -modcache
go mod download
go mod tidy

# 重新编译
make build
```

#### 2. Pod 启动失败

**问题**：K8S Pod 处于 CrashLoopBackOff

**排查**：
```bash
# 查看 Pod 状态
kubectl describe pod <pod-name>

# 查看日志
kubectl logs <pod-name> --previous

# 常见原因：
# - 配置文件错误
# - 权限不足（RBAC）
# - 镜像拉取失败
```

#### 3. 健康检查失败

**问题**：健康检查持续失败

**排查**：
```bash
# 手动测试健康检查
kubectl exec -it <pod-name> -- wget -O- http://localhost:8080/health

# 检查网络连通性
kubectl exec -it <pod-name> -- ping <service-ip>
```

#### 4. 部署超时

**问题**：部署任务长时间未完成

**排查**：
```bash
# 查看部署状态
curl http://localhost:8080/api/v1/deployments/<deployment-id>

# 查看日志
tail -f /var/log/deployment-system/app.log | grep <deployment-id>

# 可能原因：
# - 目标环境无法访问
# - 健康检查配置错误
# - 网络问题
```

### 紧急操作

#### 强制回滚

```bash
# API 方式
curl -X POST http://localhost:8080/api/v1/deployments/<deployment-id>/rollback

# CLI 方式（如果实现）
./bin/deployer rollback --service user-service --from v1.2.0 --to v1.1.0
```

#### 暂停部署

```bash
curl -X POST http://localhost:8080/api/v1/deployments/<deployment-id>/pause
```

#### 恢复部署

```bash
curl -X POST http://localhost:8080/api/v1/deployments/<deployment-id>/resume
```

---

## 升级维护

### 升级流程

#### 1. 备份配置

```bash
# Kubernetes
kubectl get configmap deployment-system-config -o yaml > backup-config.yaml

# 物理机
cp /opt/deployment-system/config/deployment.yaml backup-deployment.yaml
```

#### 2. 升级系统

**Kubernetes**:
```bash
# 更新镜像
kubectl set image deployment/intelligent-deployment-system \
  deployer=intelligent-deployment-system:v1.1.0

# 或使用新的 deployment.yaml
kubectl apply -f deployments/kubernetes/deployment.yaml
```

**物理机**:
```bash
# 停止服务
sudo systemctl stop deployment-system

# 备份旧版本
mv /opt/deployment-system/deployer /opt/deployment-system/deployer.old

# 部署新版本
cp bin/deployer /opt/deployment-system/

# 启动服务
sudo systemctl start deployment-system

# 检查状态
sudo systemctl status deployment-system
```

#### 3. 验证升级

```bash
# 检查版本
curl http://localhost:8080/version

# 检查健康状态
curl http://localhost:8080/health

# 查看日志
kubectl logs -l app=intelligent-deployment-system --tail=50
```

#### 4. 回滚（如果需要）

**Kubernetes**:
```bash
kubectl rollout undo deployment/intelligent-deployment-system
```

**物理机**:
```bash
sudo systemctl stop deployment-system
mv /opt/deployment-system/deployer.old /opt/deployment-system/deployer
sudo systemctl start deployment-system
```

### 数据迁移

如果有数据库或状态需要迁移：

```bash
# 导出数据
kubectl exec -it <pod-name> -- /app/deployer export-data > data-backup.json

# 导入数据
cat data-backup.json | kubectl exec -i <new-pod-name> -- /app/deployer import-data
```

### 维护窗口

建议在低峰时段进行维护：
- 提前通知用户
- 准备回滚方案
- 监控关键指标
- 保留详细日志

---

## 安全建议

1. **权限最小化**：只授予必要的 K8S RBAC 权限
2. **密钥管理**：使用 Kubernetes Secrets 或 Vault 存储敏感信息
3. **网络隔离**：使用 NetworkPolicy 限制 Pod 间通信
4. **镜像安全**：定期扫描 Docker 镜像漏洞
5. **审计日志**：启用 Kubernetes 审计日志
6. **访问控制**：配置 API 认证和授权

---

## 性能调优

### Kubernetes

```yaml
# 调整副本数
kubectl scale deployment intelligent-deployment-system --replicas=5

# 配置 HPA
kubectl apply -f deployments/kubernetes/hpa.yaml

# 调整资源限制
# 编辑 deployment.yaml 中的 resources 配置
```

### 物理机

```bash
# 调整 Go 运行时参数
export GOMAXPROCS=4
export GOGC=100

# 调整系统参数
sudo sysctl -w net.core.somaxconn=1024
sudo sysctl -w net.ipv4.tcp_max_syn_backlog=2048
```

---

## 支持与帮助

- **文档**: [README.md](../README.md)
- **设计文档**: [DESIGN_REQUIREMENTS.md](DESIGN_REQUIREMENTS.md)
- **Issues**: https://github.com/PonnyS/Hackathon/issues
- **Pull Request**: https://github.com/PonnyS/Hackathon/pull/2

---

**文档版本**: v1.0  
**最后更新**: 2024-10-24
