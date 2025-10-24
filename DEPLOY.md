# Jarvis 智能发布系统 - 部署指南

## 快速开始

### 方式一：使用部署脚本(推荐)

```bash
# 完整部署(构建+启动)
./deploy.sh deploy

# 或分步执行
./deploy.sh build   # 构建
./deploy.sh start   # 启动服务
```

### 方式二：使用 Makefile

```bash
# 构建后端和前端
make build

# 只构建后端
make build-backend

# 只构建前端
make build-web

# 运行服务
make run
```

### 方式三：手动构建和部署

#### 1. 构建后端

```bash
cd cmd/jarvis
go build -o jarvis jarvis.go
```

#### 2. 构建前端

```bash
cd web
npm install
npm run build
```

#### 3. 启动服务

```bash
cd cmd/jarvis
./jarvis -f etc/jarvis.yaml
```

## 配置说明

### 后端配置文件: `cmd/jarvis/etc/jarvis.yaml`

```yaml
Name: jarvis
Host: 0.0.0.0
Port: 8080

# 本地存储目录(数据持久化)
DataDir: ./data

# 可选: Kodo 文件存储配置
# Kodo:
#   AccessKey: ""
#   SecretKey: ""
#   Bucket: ""
```

### 配置项说明

- `Name`: 服务名称
- `Host`: 监听地址(0.0.0.0 表示监听所有网卡)
- `Port`: 服务端口
- `DataDir`: 数据存储目录,用于持久化发布任务、配置等数据

## 环境要求

### 后端
- Go 1.21+
- Linux/macOS/Windows 系统

### 前端
- Node.js 18+
- npm 9+

## 部署到生产环境

### 1. K8S 环境部署

创建 Deployment 和 Service:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: jarvis
spec:
  replicas: 1
  selector:
    matchLabels:
      app: jarvis
  template:
    metadata:
      labels:
        app: jarvis
    spec:
      containers:
      - name: jarvis
        image: your-registry/jarvis:latest
        ports:
        - containerPort: 8080
        volumeMounts:
        - name: data
          mountPath: /app/data
        - name: config
          mountPath: /app/etc
      volumes:
      - name: data
        persistentVolumeClaim:
          claimName: jarvis-data
      - name: config
        configMap:
          name: jarvis-config
---
apiVersion: v1
kind: Service
metadata:
  name: jarvis
spec:
  selector:
    app: jarvis
  ports:
  - port: 80
    targetPort: 8080
  type: LoadBalancer
```

### 2. 物理机/虚拟机部署

#### 使用 systemd 管理服务

创建服务文件 `/etc/systemd/system/jarvis.service`:

```ini
[Unit]
Description=Jarvis Intelligent Deployment System
After=network.target

[Service]
Type=simple
User=jarvis
WorkingDirectory=/opt/jarvis
ExecStart=/opt/jarvis/cmd/jarvis/jarvis -f /opt/jarvis/cmd/jarvis/etc/jarvis.yaml
Restart=on-failure
RestartSec=5s

[Install]
WantedBy=multi-user.target
```

启动服务:

```bash
sudo systemctl daemon-reload
sudo systemctl enable jarvis
sudo systemctl start jarvis
sudo systemctl status jarvis
```

## 数据目录结构

```
./data/
├── redis_data.json          # Redis 兼容数据
├── nodeRelease.json         # 发布任务
├── allowApps.json           # 应用白名单
├── sysParam.json            # 系统参数
├── updRecord.json           # 更新记录
├── grayNodes.json           # 灰度节点
├── nodeReleaseHistory.json  # 发布历史
└── nodeJoin.json            # 节点关联
```

## 常见问题

### 1. 端口被占用

修改 `cmd/jarvis/etc/jarvis.yaml` 中的 `Port` 配置。

### 2. 数据持久化

确保 `DataDir` 指向的目录有读写权限,并且在容器环境中挂载为持久化卷。

### 3. 前端访问 API 失败

检查前端代理配置(`web/vite.config.ts`)或者确保前端静态文件和后端服务在同一域名下。

## 监控和日志

### 查看服务日志

```bash
# 使用部署脚本
./deploy.sh status

# 使用 systemd
sudo journalctl -u jarvis -f

# 直接查看进程
ps aux | grep jarvis
```

### 健康检查

```bash
curl http://localhost:8080/v1/release/list
```

## 升级和回滚

### 升级

```bash
# 拉取最新代码
git pull

# 重新部署
./deploy.sh deploy
```

### 回滚

```bash
# 回退到指定版本
git checkout <version-tag>

# 重新部署
./deploy.sh deploy
```
