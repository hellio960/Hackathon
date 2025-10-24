# Jarvis 智能发布系统 - 前端

这是 Jarvis 智能发布系统的 Web 前端应用，基于 React + TypeScript + Ant Design 构建。

## 功能特性

### 1. 发布管理
- **创建发布任务**：支持创建灰度发布任务，可选择正式发布或功能验证
- **发布列表**：查看所有发布任务，支持按应用名称、状态筛选
- **发布详情**：查看发布任务的详细信息、配置、灰度节点和操作历史
- **继续发布**：增加灰度比例，逐步扩大发布范围
- **全量发布**：将灰度版本推广到全部节点
- **回滚**：恢复到上一个稳定版本

### 2. 应用管理
- **应用列表**：查看所有可发布的应用
- **添加应用**：注册新的应用到系统
- **编辑应用**：修改应用配置信息
- **删除应用**：移除不再使用的应用

### 3. 灰度策略
- **规则过滤**：根据节点状态、阶段等条件自动选择灰度节点
- **指定节点**：手动指定要灰度的节点列表
- **灰度比例控制**：支持 0-100% 的灰度比例设置

## 技术栈

- **React 18**：UI 框架
- **TypeScript**：类型安全
- **Ant Design 5**：UI 组件库
- **React Router 6**：路由管理
- **Axios**：HTTP 客户端
- **Vite**：构建工具
- **Day.js**：日期处理

## 快速开始

### 安装依赖

```bash
npm install
```

### 开发模式

```bash
npm run dev
```

应用将在 http://localhost:3000 启动，API 请求会自动代理到 http://localhost:8080

### 生产构建

```bash
npm run build
```

构建产物将输出到 `dist` 目录。

### 预览生产构建

```bash
npm run preview
```

## 项目结构

```
web/
├── src/
│   ├── api/                 # API 接口定义
│   │   ├── client.ts        # Axios 客户端
│   │   ├── types.ts         # TypeScript 类型定义
│   │   ├── release.ts       # 发布相关 API
│   │   └── app.ts           # 应用相关 API
│   ├── components/          # 共享组件
│   │   └── Layout/          # 布局组件
│   ├── pages/               # 页面组件
│   │   ├── Release/         # 发布管理页面
│   │   │   ├── ReleaseList.tsx      # 发布列表
│   │   │   ├── ReleaseDetail.tsx    # 发布详情
│   │   │   └── CreateRelease.tsx    # 创建发布
│   │   └── App/             # 应用管理页面
│   │       └── AppManagement.tsx    # 应用管理
│   ├── App.tsx              # 应用入口
│   ├── main.tsx             # 应用启动
│   └── index.css            # 全局样式
├── index.html               # HTML 模板
├── vite.config.ts           # Vite 配置
├── tsconfig.json            # TypeScript 配置
└── package.json             # 项目配置
```

## API 对接

前端通过 `/v1` 前缀访问后端 API：

- `POST /v1/release/create` - 创建发布任务
- `POST /v1/release/:releaseID/continue` - 继续发布
- `POST /v1/release/:releaseID/rollback` - 回滚发布
- `POST /v1/release/:releaseID/complete` - 完成发布
- `POST /v1/release/list` - 发布任务列表
- `GET /v1/release/:releaseID/detail` - 发布详情
- `GET /v1/release/:releaseID/history` - 发布历史
- `POST /v1/release/packages` - 获取可用包列表
- `GET /v1/release/allowapps` - 获取应用列表
- `POST /v1/release/allowapps` - 更新应用信息

## 配置

### 后端 API 地址

开发环境下，API 请求会通过 Vite 代理转发到 `http://localhost:8080`。

如果需要修改后端地址，编辑 `vite.config.ts`：

```typescript
export default defineConfig({
  server: {
    proxy: {
      '/v1': {
        target: 'http://your-backend-url',  // 修改为实际后端地址
        changeOrigin: true,
      },
    },
  },
})
```

生产环境下，可以通过 Nginx 等反向代理配置后端 API 路由。

## 主要功能说明

### 创建发布任务

1. 填写基本信息：设备类型、应用名称、发布说明
2. 选择发布类型：正式发布 / 功能验证
3. 选择操作类型：新增组件 / 升级组件 / 移除组件
4. 配置应用：包地址、启动命令、工作目录等
5. 设置灰度策略：选择规则过滤或指定节点，设置灰度比例
6. 提交创建

### 继续发布

在发布详情页面，点击"继续发布"按钮：
1. 输入新的灰度比例（必须 ≥ 当前比例）
2. 确认后系统会自动分配更多节点到灰度列表

### 全量发布

当灰度验证通过后，点击"全量发布"按钮：
1. 系统会将灰度版本推广到所有节点
2. 任务状态变更为"已完成"
3. 当前版本配置会更新为灰度版本

### 回滚

如果发现问题，可以点击"回滚"按钮：
1. 对于进行中的任务：清空灰度节点即可
2. 对于已完成的任务：恢复到上一个版本配置
3. 任务状态变更为"已回滚"

## 浏览器支持

- Chrome >= 90
- Firefox >= 88
- Safari >= 14
- Edge >= 90

## 开发指南

### 添加新页面

1. 在 `src/pages` 目录下创建新的页面组件
2. 在 `src/App.tsx` 中添加路由配置
3. 在 `src/components/Layout/index.tsx` 中添加菜单项（如需要）

### 添加新 API

1. 在 `src/api/types.ts` 中定义请求和响应类型
2. 在 `src/api` 目录下创建对应的 API 文件
3. 使用 `apiClient` 发起 HTTP 请求

### 代码规范

项目使用 ESLint 进行代码检查：

```bash
npm run lint
```

## 部署

### Nginx 配置示例

```nginx
server {
    listen 80;
    server_name your-domain.com;
    
    root /path/to/dist;
    index index.html;
    
    # 前端路由
    location / {
        try_files $uri $uri/ /index.html;
    }
    
    # API 代理
    location /v1/ {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }
}
```

## 常见问题

### Q: API 请求失败？
A: 检查后端服务是否正常运行在 8080 端口，查看浏览器控制台的网络请求详情。

### Q: 页面刷新后 404？
A: 这是因为使用了前端路由，需要配置服务器将所有路由请求指向 `index.html`。

### Q: 如何修改端口？
A: 编辑 `vite.config.ts` 中的 `server.port` 配置。

## License

Copyright © 2025 PCDN Team
