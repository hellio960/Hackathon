# Hackathon - Intelligent Deployment System

智能发布系统，支持 K8S 环境和物理机环境的服务部署、灰度发布、健康监控和自动回滚。

## Features

- **多环境支持**: 支持 K8S 和物理机环境部署
- **灰度发布**: 支持灵活的灰度策略和节点筛选
- **健康监控**: 自动健康检查和状态监控
- **自动回滚**: 检测到异常时自动触发回滚
- **本地存储**: 使用内存+文件存储，无需 MongoDB 和 Redis
- **独立运行**: 可直接在 Linux 系统中运行

## Quick Start

### 1. 构建

```bash
cd cmd/jarvis
make build
```

### 2. 配置

编辑配置文件 `cmd/jarvis/etc/jarvis.yaml`:

```yaml
Name: jarvis
Host: 0.0.0.0
Port: 8080

# 本地存储目录
DataDir: ./data
```

### 3. 运行

```bash
./jarvis -f etc/jarvis.yaml
```

服务将在 `http://localhost:8080` 启动。

## 本地存储说明

系统使用本地存储替代 MongoDB 和 Redis：

- **内存存储**: 数据保存在内存中，提供快速访问
- **文件持久化**: 数据自动持久化到 `DataDir` 指定的目录
- **JSON 格式**: 存储文件使用 JSON 格式，便于查看和调试

存储文件结构：
```
./data/
├── redis_data.json          # Redis 数据
├── nodeRelease.json         # 发布任务
├── allowApps.json           # 应用白名单
├── sysParam.json            # 系统参数
├── updRecord.json           # 更新记录
├── grayNodes.json           # 灰度节点
├── nodeReleaseHistory.json  # 发布历史
└── nodeJoin.json            # 节点关联
```

## API 文档

详见 [cmd/jarvis/api/](cmd/jarvis/api/) 目录下的 API 定义文件。

主要接口：

- `POST /v1/release/create` - 创建发布任务
- `POST /v1/release/continue` - 继续灰度发布
- `POST /v1/release/rollback` - 回滚发布
- `POST /v1/release/complete` - 完成发布
- `GET /v1/release/list` - 查询发布列表
- `GET /v1/release/detail` - 查询发布详情

## 架构设计

详细设计文档：
- [总体设计](doc/DESIGN.md)
- [智能发布系统](doc/INTELLIGENT_DEPLOYMENT.md)

### 核心组件

- **Executor**: 部署执行器，支持 K8S 和物理机
- **DeployManager**: 部署管理器，协调部署流程
- **HealthMonitor**: 健康监控服务，监控部署状态
- **AutoRollback**: 自动回滚服务，异常时触发回滚

## 使用示例

### 创建发布任务

```bash
curl -X POST http://localhost:8080/v1/release/create \
  -H "Content-Type: application/json" \
  -d '{
    "app": "myapp",
    "deviceType": "node",
    "releaseType": "formal",
    "opType": "update",
    "mainConfig": {
      "url": "http://example.com/myapp-v1.0.tar.gz",
      "cmd": "myapp",
      "dir": "/opt/myapp"
    },
    "alterConfig": {
      "url": "http://example.com/myapp-v1.1.tar.gz",
      "cmd": "myapp",
      "dir": "/opt/myapp"
    },
    "grayPolicy": {
      "percentage": 10
    }
  }'
```

### 查询发布状态

```bash
curl http://localhost:8080/v1/release/detail?id=<release_id>
```

### 继续灰度发布

```bash
curl -X POST http://localhost:8080/v1/release/continue \
  -H "Content-Type: application/json" \
  -d '{
    "id": "<release_id>",
    "percentage": 50
  }'
```

### 完成发布

```bash
curl -X POST http://localhost:8080/v1/release/complete \
  -H "Content-Type: application/json" \
  -d '{
    "id": "<release_id>"
  }'
```

## 开发

### 项目结构

```
.
├── cmd/
│   └── jarvis/              # 主服务
│       ├── api/             # API 定义
│       ├── internal/        # 内部实现
│       │   ├── config/      # 配置
│       │   ├── executor/    # 部署执行器
│       │   ├── handler/     # HTTP 处理器
│       │   ├── logic/       # 业务逻辑
│       │   ├── service/     # 服务组件
│       │   ├── localstorage/ # 本地存储实现
│       │   └── svc/         # 服务上下文
│       └── shared/          # 共享代码
├── common/                  # 公共库
├── sharedmodel/            # 共享数据模型
└── doc/                    # 文档
```

### 技术栈

- **框架**: go-zero
- **语言**: Go 1.21+
- **存储**: 本地文件存储（JSON）

## License

MIT
