# Jarvis 智能发布系统部署文档

## 目录

- [1. 部署架构](#1-部署架构)
- [2. 环境要求](#2-环境要求)
- [3. 快速部署](#3-快速部署)
- [4. 详细部署步骤](#4-详细部署步骤)
- [5. 配置说明](#5-配置说明)
- [6. 启动与停止](#6-启动与停止)
- [7. 监控与运维](#7-监控与运维)
- [8. 常见问题](#8-常见问题)
- [9. 生产环境部署建议](#9-生产环境部署建议)

---

## 1. 部署架构

### 1.1 系统组件

```
┌─────────────────────────────────────────────────────────┐
│                      用户访问层                          │
│                   (浏览器/CLI客户端)                      │
└─────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────┐
│                     前端代理层                           │
│    Nginx / Node.js Proxy (可选)                         │
│    - 静态文件服务                                         │
│    - API反向代理                                         │
└─────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────┐
│                     后端服务层                           │
│    Jarvis API Server (Go)                               │
│    - RESTful API                                        │
│    - 业务逻辑处理                                         │
│    - 发布任务调度                                         │
└─────────────────────────────────────────────────────────┘
                          │
           ┌──────────────┼──────────────┐
           ▼              ▼              ▼
    ┌──────────┐   ┌──────────┐   ┌──────────┐
    │ MongoDB  │   │  Redis   │   │  Kodo    │
    │ (元数据)  │   │ (缓存)    │   │ (对象存储)│
    └──────────┘   └──────────┘   └──────────┘
```

### 1.2 部署模式

#### 单机部署（推荐用于测试/开发环境）
- 所有组件部署在同一台服务器
- 资源要求：4核8G内存，100G磁盘

#### 分布式部署（推荐用于生产环境）
- 前端、后端、数据库分离部署
- 支持后端服务横向扩展
- 数据库使用集群模式

---

## 2. 环境要求

### 2.1 硬件要求

| 环境类型 | CPU | 内存 | 磁盘 | 网络 |
|---------|-----|------|------|------|
| 开发环境 | 2核 | 4GB | 50GB | 1Mbps |
| 测试环境 | 4核 | 8GB | 100GB | 10Mbps |
| 生产环境 | 8核+ | 16GB+ | 500GB+ | 100Mbps+ |

### 2.2 软件依赖

#### 必需组件

| 组件 | 版本要求 | 用途 |
|------|---------|------|
| **Go** | 1.21+ | 后端服务编译运行 |
| **MongoDB** | 4.4+ | 元数据存储 |
| **Redis** | 6.0+ | 缓存和会话存储 |

#### 可选组件

| 组件 | 版本要求 | 用途 |
|------|---------|------|
| **Node.js** | 14+ | 前端代理服务器（可用Nginx替代） |
| **Nginx** | 1.18+ | 生产环境推荐使用 |
| **Docker** | 20.10+ | 容器化部署 |

### 2.3 操作系统

- **推荐**: Linux (Ubuntu 20.04+, CentOS 7+)
- **支持**: macOS 10.15+
- **支持**: Windows 10+ (需WSL2)

---

## 3. 快速部署

### 3.1 一键构建

```bash
# 1. 克隆代码（如果还未克隆）
git clone <repository-url>
cd hackathon

# 2. 执行构建脚本
chmod +x build.sh
./build.sh

# 3. 查看构建产物
ls -lh deploy/*.tar.gz
```

### 3.2 快速启动（开发模式）

```bash
# 1. 启动依赖服务（MongoDB和Redis）
# 使用Docker快速启动
docker run -d --name mongodb -p 27017:27017 mongo:4.4
docker run -d --name redis -p 6379:6379 redis:6.0

# 2. 配置后端服务
cd cmd/jarvis
cp etc/jarvis.yaml.example etc/jarvis.yaml
# 编辑 etc/jarvis.yaml，配置MongoDB和Redis连接信息

# 3. 启动后端
go run jarvis.go -f etc/jarvis.yaml

# 4. 启动前端（新终端）
cd ../../web
node proxy.js

# 5. 访问系统
# 打开浏览器访问: http://localhost:3000
```

---

## 4. 详细部署步骤

### 4.1 准备工作

#### 4.1.1 安装Go环境

```bash
# Ubuntu/Debian
wget https://go.dev/dl/go1.21.0.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.0.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc
go version

# CentOS/RHEL
sudo yum install -y golang
go version
```

#### 4.1.2 安装MongoDB

```bash
# 使用Docker（推荐）
docker run -d \
  --name mongodb \
  -p 27017:27017 \
  -v /data/mongodb:/data/db \
  -e MONGO_INITDB_ROOT_USERNAME=admin \
  -e MONGO_INITDB_ROOT_PASSWORD=your_password \
  mongo:4.4

# 或使用包管理器安装
# Ubuntu
wget -qO - https://www.mongodb.org/static/pgp/server-4.4.asc | sudo apt-key add -
echo "deb [ arch=amd64,arm64 ] https://repo.mongodb.org/apt/ubuntu focal/mongodb-org/4.4 multiverse" | sudo tee /etc/apt/sources.list.d/mongodb-org-4.4.list
sudo apt-get update
sudo apt-get install -y mongodb-org
sudo systemctl start mongod
sudo systemctl enable mongod
```

#### 4.1.3 安装Redis

```bash
# 使用Docker（推荐）
docker run -d \
  --name redis \
  -p 6379:6379 \
  -v /data/redis:/data \
  redis:6.0 \
  redis-server --appendonly yes

# 或使用包管理器安装
# Ubuntu
sudo apt-get install -y redis-server
sudo systemctl start redis-server
sudo systemctl enable redis-server

# CentOS
sudo yum install -y redis
sudo systemctl start redis
sudo systemctl enable redis
```

#### 4.1.4 安装Node.js（可选）

```bash
# 使用nvm安装（推荐）
curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.39.0/install.sh | bash
source ~/.bashrc
nvm install 16
nvm use 16
node -v

# 或使用包管理器
# Ubuntu
curl -fsSL https://deb.nodesource.com/setup_16.x | sudo -E bash -
sudo apt-get install -y nodejs

# CentOS
curl -fsSL https://rpm.nodesource.com/setup_16.x | sudo bash -
sudo yum install -y nodejs
```

### 4.2 构建部署包

```bash
# 1. 进入项目目录
cd /path/to/hackathon

# 2. 执行构建脚本
./build.sh

# 3. 构建成功后，会生成部署包
# deploy/jarvis-release-YYYYMMDD_HHMMSS.tar.gz
```

构建脚本会自动完成：
- ✓ 检查Go环境
- ✓ 下载Go依赖
- ✓ 编译后端二进制文件
- ✓ 准备前端静态文件
- ✓ 安装Node.js依赖（如果有Node.js）
- ✓ 创建部署包

### 4.3 部署到服务器

#### 4.3.1 传输部署包

```bash
# 将部署包传输到目标服务器
scp deploy/jarvis-release-*.tar.gz user@server:/opt/

# 登录目标服务器
ssh user@server

# 解压部署包
cd /opt
tar -xzf jarvis-release-*.tar.gz
cd jarvis-release-*
```

#### 4.3.2 配置后端服务

```bash
cd backend

# 复制配置文件
cp jarvis.yaml.example jarvis.yaml

# 编辑配置文件
vim jarvis.yaml
```

修改以下配置项：

```yaml
Name: jarvis
Host: 0.0.0.0        # 监听地址
Port: 8100           # 监听端口

# MongoDB配置
Mongo:
  Url: mongodb://admin:password@localhost:27017/jarvis?authSource=admin

# Redis配置
BizRedisConfig:
  Host: localhost:6379
  Pass: ""           # 如有密码请填写

# Kodo对象存储配置
Kodo:
  AccessKey: "your-access-key"
  SecretKey: "your-secret-key"
  Bucket: "your-bucket-name"

# 缓存配置
CacheConfig:
  - Host: localhost:6379
    Type: node
    Pass: ""         # 如有密码请填写
```

#### 4.3.3 配置前端（方案一：使用Node.js代理）

```bash
cd ../frontend

# 如果还没有安装依赖
npm install

# 编辑代理配置（如需要）
vim proxy.js
```

修改代理目标地址：

```javascript
const apiProxy = httpProxy.createProxyMiddleware({
    target: 'http://127.0.0.1:8100',  // 后端API地址
    changeOrigin: true,
    // ...
});
```

#### 4.3.4 配置前端（方案二：使用Nginx）

```bash
# 安装Nginx
sudo apt-get install -y nginx

# 创建Nginx配置
sudo vim /etc/nginx/sites-available/jarvis
```

Nginx配置示例：

```nginx
server {
    listen 80;
    server_name your-domain.com;  # 修改为你的域名

    # 前端静态文件
    location / {
        root /opt/jarvis-release-*/frontend;
        index release-list.html;
        try_files $uri $uri/ /release-list.html;
    }

    # API反向代理
    location /api/ {
        proxy_pass http://127.0.0.1:8100/;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        
        # 支持WebSocket（如需要）
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }
}
```

启用配置：

```bash
# 创建软链接
sudo ln -s /etc/nginx/sites-available/jarvis /etc/nginx/sites-enabled/

# 测试配置
sudo nginx -t

# 重启Nginx
sudo systemctl restart nginx
```

### 4.4 初始化数据库

```bash
# 创建数据库和初始数据
mongo mongodb://admin:password@localhost:27017/admin

# 在MongoDB shell中执行
use jarvis

# 创建索引（后端启动时会自动创建，也可手动创建）
db.nodeRelease.createIndex({"service": 1, "version": 1})
db.nodeRelease.createIndex({"status": 1})
db.allowApps.createIndex({"name": 1}, {unique: true})
```

---

## 5. 配置说明

### 5.1 后端配置文件详解

`backend/jarvis.yaml` 配置项说明：

```yaml
# 服务基础配置
Name: jarvis                    # 服务名称
Host: 0.0.0.0                   # 监听地址，0.0.0.0表示监听所有网卡
Port: 8100                      # HTTP端口

# 日志配置
Log:
  Level: info                   # 日志级别: debug/info/warn/error
  Mode: file                    # 日志模式: console/file
  Path: logs                    # 日志文件路径
  KeepDays: 7                   # 日志保留天数

# MongoDB配置
Mongo:
  Url: mongodb://[username:password@]host:port/database[?options]
  # 示例:
  # - 无认证: mongodb://localhost:27017/jarvis
  # - 有认证: mongodb://admin:pass@localhost:27017/jarvis?authSource=admin
  # - 副本集: mongodb://host1:27017,host2:27017,host3:27017/jarvis?replicaSet=rs0

# 业务Redis配置（用于灰度节点缓存）
BizRedisConfig:
  Host: 127.0.0.1:6379          # Redis地址
  Pass: ""                      # Redis密码（可选）
  Type: node                    # Redis类型
  Db: 0                         # 数据库编号

# Kodo对象存储配置（七牛云存储）
Kodo:
  AccessKey: "your-ak"          # 七牛云AccessKey
  SecretKey: "your-sk"          # 七牛云SecretKey
  Bucket: "your-bucket"         # 存储空间名称
  Domain: "your-domain.com"     # CDN加速域名（可选）

# 缓存配置（用于节点信息缓存）
CacheConfig:
  - Host: 127.0.0.1:6379
    Type: node
    Pass: ""
    Db: 1                       # 使用不同的DB避免冲突

# 超时配置（可选）
Timeout: 30000                  # API超时时间（毫秒）

# CORS配置（可选）
Cors:
  AllowOrigins:
    - "*"                       # 允许的来源，生产环境建议配置具体域名
  AllowMethods:
    - GET
    - POST
    - PUT
    - DELETE
    - OPTIONS
  AllowHeaders:
    - Content-Type
    - Authorization
```

### 5.2 前端代理配置

`frontend/proxy.js` 关键配置：

```javascript
// 监听端口
const PORT = process.env.PORT || 3000;

// 后端API地址
const apiProxy = httpProxy.createProxyMiddleware({
    target: 'http://127.0.0.1:8100',  // 修改为实际后端地址
    changeOrigin: true,
    pathRewrite: {
        '^/api': ''                    // URL重写规则
    }
});
```

### 5.3 环境变量配置

支持通过环境变量覆盖配置：

```bash
# 后端服务
export JARVIS_PORT=8100
export MONGO_URL="mongodb://localhost:27017/jarvis"
export REDIS_HOST="localhost:6379"

# 前端代理
export PORT=3000
export API_TARGET="http://localhost:8100"
```

---

## 6. 启动与停止

### 6.1 使用启动脚本

#### 启动服务

```bash
cd /opt/jarvis-release-*
./start.sh
```

启动脚本会自动：
- ✓ 检查配置文件
- ✓ 启动后端服务
- ✓ 启动前端代理（如有Node.js）
- ✓ 显示服务状态

#### 停止服务

```bash
./stop.sh
```

停止脚本会优雅关闭所有服务。

### 6.2 手动启动

#### 启动后端

```bash
cd backend

# 前台运行（用于调试）
./jarvis -f jarvis.yaml

# 后台运行
nohup ./jarvis -f jarvis.yaml > jarvis.log 2>&1 &

# 查看日志
tail -f jarvis.log
```

#### 启动前端代理

```bash
cd frontend

# 前台运行
node proxy.js

# 后台运行
nohup node proxy.js > proxy.log 2>&1 &

# 查看日志
tail -f proxy.log
```

### 6.3 使用systemd管理服务

#### 创建后端服务文件

```bash
sudo vim /etc/systemd/system/jarvis-backend.service
```

内容如下：

```ini
[Unit]
Description=Jarvis Backend Service
After=network.target mongod.service redis.service

[Service]
Type=simple
User=jarvis
WorkingDirectory=/opt/jarvis-release/backend
ExecStart=/opt/jarvis-release/backend/jarvis -f jarvis.yaml
Restart=on-failure
RestartSec=5s

[Install]
WantedBy=multi-user.target
```

#### 创建前端服务文件

```bash
sudo vim /etc/systemd/system/jarvis-frontend.service
```

内容如下：

```ini
[Unit]
Description=Jarvis Frontend Proxy
After=network.target

[Service]
Type=simple
User=jarvis
WorkingDirectory=/opt/jarvis-release/frontend
ExecStart=/usr/bin/node proxy.js
Restart=on-failure
RestartSec=5s

[Install]
WantedBy=multi-user.target
```

#### 管理服务

```bash
# 重载systemd配置
sudo systemctl daemon-reload

# 启动服务
sudo systemctl start jarvis-backend
sudo systemctl start jarvis-frontend

# 设置开机自启
sudo systemctl enable jarvis-backend
sudo systemctl enable jarvis-frontend

# 查看服务状态
sudo systemctl status jarvis-backend
sudo systemctl status jarvis-frontend

# 查看日志
sudo journalctl -u jarvis-backend -f
sudo journalctl -u jarvis-frontend -f

# 停止服务
sudo systemctl stop jarvis-backend
sudo systemctl stop jarvis-frontend
```

---

## 7. 监控与运维

### 7.1 健康检查

#### 后端健康检查

```bash
# 检查服务是否运行
curl http://localhost:8100/health

# 预期响应
{
  "status": "ok",
  "timestamp": "2024-01-01T00:00:00Z"
}
```

#### 进程监控

```bash
# 查看后端进程
ps aux | grep jarvis

# 查看端口监听
netstat -tlnp | grep 8100

# 查看资源占用
top -p $(pgrep jarvis)
```

### 7.2 日志管理

#### 日志位置

```
backend/
  ├── jarvis.log          # 后端主日志
  ├── logs/               # 详细日志目录
  │   ├── access.log      # 访问日志
  │   ├── error.log       # 错误日志
  │   └── stat.log        # 统计日志

frontend/
  └── proxy.log           # 前端代理日志
```

#### 日志查看

```bash
# 实时查看日志
tail -f backend/jarvis.log

# 查看最近100行
tail -n 100 backend/jarvis.log

# 搜索错误日志
grep "ERROR" backend/jarvis.log

# 按时间查看日志
grep "2024-01-01" backend/jarvis.log
```

#### 日志轮转

使用logrotate自动轮转日志：

```bash
sudo vim /etc/logrotate.d/jarvis
```

配置内容：

```
/opt/jarvis-release/backend/logs/*.log {
    daily
    rotate 30
    compress
    delaycompress
    notifempty
    missingok
    create 0644 jarvis jarvis
    sharedscripts
    postrotate
        systemctl reload jarvis-backend > /dev/null 2>&1 || true
    endscript
}
```

### 7.3 性能监控

#### 系统资源监控

```bash
# CPU和内存使用
top -p $(pgrep jarvis)

# 磁盘使用
df -h

# 网络连接
ss -antp | grep 8100
```

#### 数据库监控

```bash
# MongoDB状态
mongo --eval "db.serverStatus()"

# Redis状态
redis-cli info

# 查看连接数
mongo --eval "db.currentOp()"
```

#### 应用监控

建议使用以下监控工具：
- **Prometheus + Grafana**: 指标采集和可视化
- **ELK Stack**: 日志聚合和分析
- **Sentry**: 错误跟踪

### 7.4 备份策略

#### MongoDB备份

```bash
# 创建备份脚本
cat > /opt/backup/backup-mongodb.sh <<'EOF'
#!/bin/bash
BACKUP_DIR="/opt/backup/mongodb"
DATE=$(date +%Y%m%d_%H%M%S)
mkdir -p $BACKUP_DIR
mongodump --uri="mongodb://admin:password@localhost:27017/jarvis?authSource=admin" --out=$BACKUP_DIR/$DATE
tar -czf $BACKUP_DIR/jarvis-mongo-$DATE.tar.gz -C $BACKUP_DIR $DATE
rm -rf $BACKUP_DIR/$DATE
find $BACKUP_DIR -name "*.tar.gz" -mtime +7 -delete
EOF

chmod +x /opt/backup/backup-mongodb.sh

# 添加到crontab（每天凌晨2点备份）
crontab -e
0 2 * * * /opt/backup/backup-mongodb.sh
```

#### Redis备份

```bash
# Redis会自动生成RDB文件
# 定期复制RDB文件到备份目录
cat > /opt/backup/backup-redis.sh <<'EOF'
#!/bin/bash
BACKUP_DIR="/opt/backup/redis"
DATE=$(date +%Y%m%d_%H%M%S)
mkdir -p $BACKUP_DIR
cp /var/lib/redis/dump.rdb $BACKUP_DIR/dump-$DATE.rdb
find $BACKUP_DIR -name "dump-*.rdb" -mtime +7 -delete
EOF

chmod +x /opt/backup/backup-redis.sh
crontab -e
0 3 * * * /opt/backup/backup-redis.sh
```

---

## 8. 常见问题

### 8.1 后端启动失败

#### 问题1: 端口被占用

```bash
# 错误信息
bind: address already in use

# 解决方案
# 1. 查找占用端口的进程
lsof -i :8100

# 2. 杀死占用进程
kill -9 <PID>

# 3. 或修改配置文件中的端口
vim backend/jarvis.yaml
```

#### 问题2: MongoDB连接失败

```bash
# 错误信息
connection refused / authentication failed

# 解决方案
# 1. 检查MongoDB是否运行
sudo systemctl status mongod

# 2. 检查连接字符串
# 确保用户名、密码、数据库名正确
mongodb://username:password@host:port/database?authSource=admin

# 3. 检查MongoDB日志
sudo tail -f /var/log/mongodb/mongod.log

# 4. 测试连接
mongo "mongodb://admin:password@localhost:27017/jarvis?authSource=admin"
```

#### 问题3: Redis连接失败

```bash
# 解决方案
# 1. 检查Redis是否运行
sudo systemctl status redis

# 2. 测试连接
redis-cli ping

# 3. 如果设置了密码
redis-cli -a your_password ping

# 4. 检查防火墙
sudo ufw status
sudo ufw allow 6379
```

### 8.2 前端访问问题

#### 问题1: 页面无法访问

```bash
# 解决方案
# 1. 检查前端服务是否运行
ps aux | grep node
ps aux | grep nginx

# 2. 检查端口监听
netstat -tlnp | grep 3000
netstat -tlnp | grep 80

# 3. 检查防火墙
sudo ufw allow 3000
sudo ufw allow 80
```

#### 问题2: API调用失败 (CORS错误)

```javascript
// 在 proxy.js 中添加CORS头部
app.use((req, res, next) => {
    res.header('Access-Control-Allow-Origin', '*');
    res.header('Access-Control-Allow-Methods', 'GET,PUT,POST,DELETE,OPTIONS');
    res.header('Access-Control-Allow-Headers', 'Content-Type, Authorization');
    if (req.method === 'OPTIONS') {
        res.sendStatus(200);
    } else {
        next();
    }
});
```

#### 问题3: 静态资源404

```bash
# 检查文件路径
ls -la frontend/

# 检查Nginx配置
sudo nginx -t
sudo tail -f /var/log/nginx/error.log
```

### 8.3 性能问题

#### 问题1: 响应慢

```bash
# 1. 检查系统资源
top
free -m
df -h

# 2. 检查数据库性能
# MongoDB慢查询
db.setProfilingLevel(2)
db.system.profile.find().limit(10).sort({ts:-1})

# 3. 检查Redis性能
redis-cli --latency

# 4. 优化建议
# - 增加MongoDB索引
# - 调整Redis缓存策略
# - 升级服务器配置
```

#### 问题2: 内存占用高

```bash
# 1. 检查进程内存
ps aux --sort=-%mem | head

# 2. MongoDB内存优化
# 修改 /etc/mongod.conf
storage:
  wiredTiger:
    engineConfig:
      cacheSizeGB: 2  # 限制缓存大小

# 3. Go程序内存优化
GOGC=100  # 调整GC频率
```

### 8.4 数据问题

#### 问题1: 数据丢失

```bash
# 恢复MongoDB备份
mongorestore --uri="mongodb://admin:password@localhost:27017" --db=jarvis /opt/backup/mongodb/latest/jarvis

# 恢复Redis备份
redis-cli SHUTDOWN SAVE
cp /opt/backup/redis/dump-latest.rdb /var/lib/redis/dump.rdb
sudo systemctl start redis
```

#### 问题2: 数据不一致

```bash
# 清理Redis缓存
redis-cli FLUSHDB

# 重启后端服务重新加载数据
sudo systemctl restart jarvis-backend
```

---

## 9. 生产环境部署建议

### 9.1 安全加固

#### 防火墙配置

```bash
# 只开放必要端口
sudo ufw default deny incoming
sudo ufw default allow outgoing
sudo ufw allow 22/tcp      # SSH
sudo ufw allow 80/tcp      # HTTP
sudo ufw allow 443/tcp     # HTTPS
sudo ufw enable

# 内网服务不对外开放
# MongoDB: 127.0.0.1:27017
# Redis: 127.0.0.1:6379
# Backend API: 127.0.0.1:8100
```

#### HTTPS配置

使用Let's Encrypt免费SSL证书：

```bash
# 安装certbot
sudo apt-get install -y certbot python3-certbot-nginx

# 获取证书
sudo certbot --nginx -d your-domain.com

# 自动续期
sudo certbot renew --dry-run
```

Nginx HTTPS配置：

```nginx
server {
    listen 443 ssl http2;
    server_name your-domain.com;
    
    ssl_certificate /etc/letsencrypt/live/your-domain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/your-domain.com/privkey.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;
    
    # ... 其他配置
}

server {
    listen 80;
    server_name your-domain.com;
    return 301 https://$server_name$request_uri;
}
```

#### 数据库安全

```bash
# MongoDB启用认证
mongo
> use admin
> db.createUser({
    user: "admin",
    pwd: "strong_password",
    roles: ["root"]
})

# 修改 /etc/mongod.conf
security:
  authorization: enabled

# Redis设置密码
# 修改 /etc/redis/redis.conf
requirepass strong_password
```

### 9.2 高可用部署

#### 后端服务负载均衡

```nginx
# Nginx负载均衡配置
upstream jarvis_backend {
    least_conn;
    server 192.168.1.10:8100 weight=1 max_fails=3 fail_timeout=30s;
    server 192.168.1.11:8100 weight=1 max_fails=3 fail_timeout=30s;
    server 192.168.1.12:8100 weight=1 max_fails=3 fail_timeout=30s;
}

server {
    location /api/ {
        proxy_pass http://jarvis_backend/;
        proxy_next_upstream error timeout http_502 http_503 http_504;
        # ... 其他配置
    }
}
```

#### MongoDB副本集

```bash
# 配置副本集
# 在三台服务器上分别启动MongoDB实例
mongod --replSet rs0 --bind_ip localhost,<server_ip> --port 27017

# 初始化副本集（在主节点）
mongo
> rs.initiate({
    _id: "rs0",
    members: [
        { _id: 0, host: "192.168.1.10:27017" },
        { _id: 1, host: "192.168.1.11:27017" },
        { _id: 2, host: "192.168.1.12:27017" }
    ]
})

# 修改后端配置
Mongo:
  Url: mongodb://192.168.1.10:27017,192.168.1.11:27017,192.168.1.12:27017/jarvis?replicaSet=rs0
```

#### Redis哨兵模式

```bash
# Redis哨兵配置
# /etc/redis/sentinel.conf
sentinel monitor mymaster 192.168.1.10 6379 2
sentinel down-after-milliseconds mymaster 5000
sentinel parallel-syncs mymaster 1
sentinel failover-timeout mymaster 10000

# 启动哨兵
redis-sentinel /etc/redis/sentinel.conf
```

### 9.3 容器化部署

#### Docker Compose配置

创建 `docker-compose.yml`:

```yaml
version: '3.8'

services:
  mongodb:
    image: mongo:4.4
    container_name: jarvis-mongodb
    restart: always
    environment:
      MONGO_INITDB_ROOT_USERNAME: admin
      MONGO_INITDB_ROOT_PASSWORD: ${MONGO_PASSWORD}
    volumes:
      - mongodb_data:/data/db
    networks:
      - jarvis-network

  redis:
    image: redis:6.0
    container_name: jarvis-redis
    restart: always
    command: redis-server --requirepass ${REDIS_PASSWORD}
    volumes:
      - redis_data:/data
    networks:
      - jarvis-network

  backend:
    image: jarvis-backend:latest
    container_name: jarvis-backend
    restart: always
    depends_on:
      - mongodb
      - redis
    environment:
      - MONGO_URL=mongodb://admin:${MONGO_PASSWORD}@mongodb:27017/jarvis?authSource=admin
      - REDIS_HOST=redis:6379
      - REDIS_PASS=${REDIS_PASSWORD}
    ports:
      - "8100:8100"
    networks:
      - jarvis-network

  frontend:
    image: nginx:alpine
    container_name: jarvis-frontend
    restart: always
    depends_on:
      - backend
    volumes:
      - ./web:/usr/share/nginx/html
      - ./nginx.conf:/etc/nginx/conf.d/default.conf
    ports:
      - "80:80"
      - "443:443"
    networks:
      - jarvis-network

volumes:
  mongodb_data:
  redis_data:

networks:
  jarvis-network:
    driver: bridge
```

启动容器：

```bash
# 设置环境变量
export MONGO_PASSWORD=your_mongo_password
export REDIS_PASSWORD=your_redis_password

# 启动所有服务
docker-compose up -d

# 查看日志
docker-compose logs -f

# 停止服务
docker-compose down
```

### 9.4 监控告警

#### Prometheus配置

```yaml
# prometheus.yml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'jarvis-backend'
    static_configs:
      - targets: ['localhost:8100']
    metrics_path: /metrics

  - job_name: 'mongodb'
    static_configs:
      - targets: ['localhost:9216']

  - job_name: 'redis'
    static_configs:
      - targets: ['localhost:9121']
```

#### Grafana仪表盘

导入预配置的仪表盘ID:
- Go应用: 10826
- MongoDB: 2583
- Redis: 763
- Nginx: 9614

#### 告警规则

```yaml
# alerting_rules.yml
groups:
  - name: jarvis_alerts
    rules:
      - alert: BackendDown
        expr: up{job="jarvis-backend"} == 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "后端服务宕机"
          description: "Jarvis后端服务已停止运行"

      - alert: HighMemoryUsage
        expr: process_resident_memory_bytes{job="jarvis-backend"} > 1e9
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "内存使用率过高"
          description: "后端服务内存使用超过1GB"

      - alert: DatabaseConnectionFailed
        expr: mongodb_up == 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "数据库连接失败"
          description: "无法连接到MongoDB"
```

---

## 10. 附录

### 10.1 端口列表

| 服务 | 默认端口 | 说明 |
|------|---------|------|
| 后端API | 8100 | RESTful API服务 |
| 前端代理 | 3000 | Node.js代理服务器 |
| Nginx | 80/443 | HTTP/HTTPS服务 |
| MongoDB | 27017 | 数据库服务 |
| Redis | 6379 | 缓存服务 |

### 10.2 目录结构

```
jarvis-release/
├── backend/                    # 后端服务
│   ├── jarvis                 # 可执行文件
│   ├── jarvis.yaml            # 配置文件
│   └── jarvis.log             # 运行日志
├── frontend/                   # 前端文件
│   ├── *.html                 # HTML页面
│   ├── css/                   # 样式文件
│   ├── js/                    # JavaScript文件
│   ├── proxy.js               # 代理服务器
│   └── package.json           # 依赖配置
├── doc/                       # 文档
│   ├── DESIGN.md              # 设计文档
│   └── DEPLOY.md              # 部署文档（本文件）
├── start.sh                   # 启动脚本
├── stop.sh                    # 停止脚本
└── README.md                  # 说明文件
```

### 10.3 相关链接

- **项目仓库**: [GitHub链接]
- **问题反馈**: [Issue链接]
- **技术文档**: 
  - [Go-Zero框架](https://go-zero.dev/)
  - [MongoDB文档](https://docs.mongodb.com/)
  - [Redis文档](https://redis.io/documentation)

### 10.4 联系方式

如有问题，请通过以下方式联系：
- 提交Issue到GitHub仓库
- 发送邮件至团队邮箱
- 加入项目技术交流群

---

**文档版本**: v1.0  
**最后更新**: 2024-10-25  
**适用版本**: Jarvis v1.1+
