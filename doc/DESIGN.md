# 智能发布系统详细设计文档

## 1. 项目概述

### 1.1 项目背景

在现代分布式系统中，服务的版本发布是一个高风险操作。传统的全量发布方式存在以下痛点：

- **风险高**：一次性全量发布，一旦新版本存在问题，影响范围大
- **回滚困难**：发现问题后回滚耗时长，业务中断时间不可控
- **缺乏验证**：新版本上线前缺少小范围验证机制
- **环境差异**：K8S 和物理机环境发布逻辑不统一，维护成本高
- **并发冲突**：多版本发布时容易产生冲突，节点分配混乱

### 1.2 项目目标

本系统旨在构建一个**智能化、自动化的灰度发布系统**，具备以下核心能力：

1. **灰度发布（Canary Deployment）**：支持按比例、按节点、按规则的渐进式发布
2. **自动故障检测与回滚**：实时监控服务健康状态，异常时自动回滚
3. **多版本并行发布**：允许同一服务的不同版本并行发布（如 v1.1 未完成时可启动 v1.2）
4. **环境统一抽象**：用同一套逻辑支持 K8S 和物理机环境
5. **精细化管控**：支持节点级别的灰度控制和流量管理

### 1.3 核心价值

- **降低发布风险**：通过灰度发布，将影响范围控制在最小
- **提升发布效率**：自动化流程减少人工干预，支持并行发布
- **保障业务连续性**：自动回滚机制确保问题快速恢复
- **统一运维体验**：屏蔽底层环境差异，提供一致的发布体验

## 2. 系统架构

### 2.1 整体架构

系统采用**分层架构设计**，从上到下分为：

```
┌─────────────────────────────────────────────────────────────┐
│                      API 层 (RESTful API)                    │
│           /v1/release/create, /continue, /rollback...       │
└─────────────────────────────────────────────────────────────┘
                              │
┌─────────────────────────────────────────────────────────────┐
│                      业务逻辑层 (Logic Layer)                │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │ 发布创建逻辑  │  │ 发布继续逻辑  │  │ 回滚逻辑      │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
└─────────────────────────────────────────────────────────────┘
                              │
┌─────────────────────────────────────────────────────────────┐
│                      数据模型层 (Model Layer)                │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │NodeRelease   │  │GrayNodes     │  │ReleaseHistory│      │
│  │  Model       │  │   Model      │  │   Model      │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
└─────────────────────────────────────────────────────────────┘
                              │
┌─────────────────────────────────────────────────────────────┐
│                      存储层 (Storage Layer)                  │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │  MongoDB     │  │    Redis     │  │   Kodo(OSS)  │      │
│  │(元数据存储)   │  │(节点缓存)     │  │(包存储)       │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
└─────────────────────────────────────────────────────────────┘
```

### 2.2 技术架构

```
┌──────────────────────────────────────────────────────────────┐
│                        前端/客户端                            │
│                  (Web UI / CLI / API Client)                 │
└──────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌──────────────────────────────────────────────────────────────┐
│                      Go-Zero 框架                             │
│  ┌────────────────────────────────────────────────────────┐  │
│  │              HTTP Server (RESTful)                     │  │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐            │  │
│  │  │ Handler  │→ │  Logic   │→ │  Model   │            │  │
│  │  └──────────┘  └──────────┘  └──────────┘            │  │
│  └────────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────────┘
                              │
              ┌───────────────┼───────────────┐
              ▼               ▼               ▼
    ┌─────────────┐  ┌─────────────┐  ┌─────────────┐
    │  MongoDB    │  │    Redis    │  │  Kodo SDK   │
    │  (元数据)    │  │  (灰度节点)  │  │  (包管理)    │
    └─────────────┘  └─────────────┘  └─────────────┘
```

### 2.3 架构特点

1. **分层解耦**：API、逻辑、数据三层分离，职责清晰
2. **模型驱动**：以数据模型为核心，统一数据访问接口
3. **缓存加速**：Redis 缓存灰度节点，减少数据库压力
4. **对象存储**：Kodo 统一管理发布包，支持版本追溯
5. **可扩展性**：基于接口编程，便于扩展新的发布策略

## 3. 核心模块设计

### 3.1 组件管理 (AllowApps)

#### 3.1.1 功能概述

组件管理模块用于管理可发布的应用组件,提供组件的增删改查功能。每个组件关联特定的节点类型,并指定其在 Kodo 对象存储中的存放路径。

#### 3.1.2 数据模型

```go
type AllowAppsTemplate struct {
    ID       string    // 组件 ID
    Name     string    // 组件名称
    NodeType string    // 节点类型
    Path     string    // Kodo 中的存放路径
    Desc     string    // 组件描述
    Operator string    // 操作人
    CreateAt time.Time // 创建时间
    UpdateAt time.Time // 更新时间
}
```

#### 3.1.3 核心功能

- **组件注册**:新增可发布的应用组件
- **组件查询**:按名称、节点类型搜索组件
- **组件更新**:修改组件路径和描述信息
- **组件删除**:移除不再使用的组件

#### 3.1.4 应用场景

- 在创建发布任务前,先在组件管理中注册应用
- 组件的 Path 字段指向 Kodo 中该应用所有版本包的存放目录
- 发布时从该目录中选择具体的版本包进行发布

### 3.2 发布任务管理 (NodeRelease)

#### 3.2.1 数据模型

```go
type NodeRelease struct {
    ID          string            // 任务 ID
    App         string            // 应用名称
    DeviceType  string            // 设备类型
    ReleaseType ReleaseType       // 发布类型：formal(正式)/beta(测试)
    OpType      NodeReleaseOpType // 操作类型：add/update/delete
    MainConfig  *AppConfig        // 当前主版本配置
    AlterConfig *AppConfig        // 灰度版本配置
    GrayPolicy  GrayPolicy        // 灰度策略
    State       NodeReleaseState  // 状态：processing/complete/rollbacked
    Operator    string            // 操作人
    Describe    string            // 发布说明
    CreateAt    time.Time         // 创建时间
    UpdateAt    time.Time         // 更新时间
    EndAt       time.Time         // 结束时间
}
```

#### 3.2.2 发布类型

- **正式发布 (formal)**：生产环境的版本升级，同一应用同一设备类型只能有一个正式发布任务在进行中
- **功能验证 (beta)**：新功能的小范围验证，可与正式发布并行

#### 3.2.3 操作类型

- **新增组件 (add)**：部署新应用
- **升级组件 (update)**：升级现有应用版本
- **移除组件 (delete)**：下线应用

#### 3.2.4 状态流转

```
          创建
           ▼
    ┌──────────────┐
    │  processing  │ ◄─┐
    │  (进行中)     │   │ 继续发布
    └──────────────┘   │
           │           │
      ┌────┴────┐      │
      ▼         ▼      │
 ┌─────────┐ ┌────────┴───┐
 │complete │ │ rollbacked │
 │(已完成) │ │  (已回滚)   │
 └─────────┘ └────────────┘
      │
      ▼ (可回滚)
 ┌────────────┐
 │ rollbacked │
 │  (已回滚)   │
 └────────────┘
```

### 3.3 灰度策略 (GrayPolicy)

#### 3.3.1 数据模型

```go
type GrayPolicy struct {
    NodeIds    []string    // 指定灰度节点 ID 列表
    Filter     *GrayFilter // 灰度节点过滤规则
    Percentage int         // 灰度比例 (0-100)
    StatInfo   *StatInfo   // 统计信息
}

type GrayFilter struct {
    DevType     string   // 设备类型
    CustomerIds []uint32 // 客户 ID 列表
    Status      string   // 节点状态：online/offline
    Stages      []string // 节点阶段标识
}

type StatInfo struct {
    MarkID             string    // 渐进查找标记
    TotalCount         int64     // 符合条件的总节点数
    AllowCount         int64     // 当前灰度中的节点数
    TotalCountUpdateAt time.Time // 总数更新时间
    AllowCountUpdateAt time.Time // 灰度数更新时间
}
```

#### 3.3.2 灰度模式

**1. 指定节点模式**

- 通过 `NodeIds` 字段明确指定灰度节点
- 适用场景：小范围验证、特定客户测试
- 优先级：最高

**2. 规则过滤模式**

- 通过 `Filter` 字段设置过滤条件
- 自动从符合条件的节点池中按 `Percentage` 比例选择节点
- 适用场景：大规模灰度、按业务/地域分批发布

**3. 混合模式**

- 同时设置 `NodeIds` 和 `Filter`
- 先满足指定节点，再按规则补充

#### 3.3.3 灰度比例控制

- **初始比例**：创建任务时设定初始灰度比例（如 10%）
- **渐进增加**：通过 `/continue` 接口逐步增加比例（如 10% → 30% → 50% → 100%）
- **动态调整**：支持增加/删除特定节点

### 3.4 应用配置 (AppConfig)

```go
type AppConfig struct {
    PackageUrl  string   // 包下载地址 (Kodo URL)
    PackageType string   // 包类型：tar.gz, zip, etc.
    Cmd         string   // 启动命令
    Args        []string // 启动参数
    WorkDir     string   // 工作目录
    HealthUrl   string   // 健康检查 URL
    PackageMd5  string   // 包 MD5 校验和
}
```

- **MainConfig**：当前线上运行的版本配置
- **AlterConfig**：灰度中的新版本配置

### 3.5 灰度节点管理 (GrayNodes)

#### 3.5.1 数据模型

```go
type GrayNode struct {
    ID        string    // 记录 ID
    ReleaseID string    // 所属发布任务 ID
    NodeId    string    // 节点 ID
    CreateAt  time.Time // 加入灰度时间
}
```

#### 3.5.2 存储设计

- **持久化存储**：MongoDB 存储灰度节点记录，用于历史追溯
- **缓存层**：Redis 存储当前灰度中的节点集合，格式：`Set<NodeId>`
- **Topic 机制**：每个应用的灰度节点存储在独立的 Redis Key 中

```
Key Pattern: 
- 大节点: jarvis_upd_config_allow_nodes_{app}_{releaseID}
- 小盒子: box_upd_config_allow_nodes_{app}_{releaseID}
Value: Set<NodeId>

注: 实际实现中使用固定前缀而非动态的 nodeType 占位符
```

#### 3.5.3 节点分配策略

**冲突检测**：
- 同一节点不能同时加入多个正在进行的发布任务
- 新增节点前检查是否已在其他任务的灰度列表中

**自动分配算法**（规则过滤模式）：
1. 根据 `GrayFilter` 从全量节点池中筛选符合条件的节点
2. 计算目标灰度节点数：`TotalCount * Percentage / 100`
3. 从候选节点中随机/顺序选择节点加入灰度
   - 推荐使用 MongoDB `$sample` 聚合提升性能:
     ```javascript
     db.nodes.aggregate([
       { $match: filterCriteria },
       { $sample: { size: targetCount } }
     ])
     ```
   - 复杂度从 O(N + M) 降至 O(log N + M)
4. 更新 Redis 和 MongoDB

### 3.6 节点搜索 (NodesSearch)

#### 3.6.1 功能概述

节点搜索模块提供根据多种条件查询节点的能力,支持按设备类型、阶段、状态、业务 ID 等维度进行灵活组合查询。

#### 3.6.2 查询条件

```go
type NodesSearchCond struct {
    DevType     string   // 设备类型
    Stage       string   // 节点阶段
    Status      string   // 节点状态:online/outline
    CustomerIds []uint32 // 业务 ID 列表
    NodeIds     []string // 指定节点 ID
    Size        int      // 返回数量
}
```

#### 3.6.3 应用场景

- 在创建发布任务时,通过节点搜索预览符合条件的节点
- 验证灰度策略的过滤规则是否正确
- 为灰度发布选择合适的目标节点

### 3.7 节点模拟器 (Node Simulator)

#### 3.7.1 功能概述

节点模拟器是一个测试工具,用于模拟节点的行为和状态,便于在开发和测试环境中验证发布系统的功能。

#### 3.7.2 核心功能

- **节点模拟**:创建虚拟节点用于测试
- **状态模拟**:模拟节点的在线/离线状态
- **配置验证**:验证节点是否正确接收到发布配置
- **批量测试**:支持批量创建和管理测试节点

#### 3.7.3 使用场景

- 发布系统功能测试
- 灰度策略验证
- 多节点并发场景模拟
- 发布流程演示

### 3.8 发布历史 (ReleaseHistory)

#### 3.8.1 数据模型

```go
type NodeReleaseHistory struct {
    ID          string
    ReleaseID   string
    Operation   string            // 操作类型：create/continue/complete/rollback
    Operator    string            // 操作人
    OpTime      time.Time         // 操作时间
    BeforeState NodeReleaseState  // 操作前状态
    AfterState  NodeReleaseState  // 操作后状态
    GrayPolicyInfo *GrayPolicyRemark  // 灰度策略变更
    AppConfigInfo  *AppConfigRemark   // 配置变更
    CreateAt    time.Time
    UpdateAt    time.Time
}
```

#### 3.8.2 用途

- **审计日志**：记录所有发布操作，满足合规要求
- **问题排查**：出现问题时可追溯操作历史
- **数据分析**：统计发布成功率、回滚率等指标

## 4. 核心流程设计

### 4.1 创建发布任务流程

```
[客户端] → POST /v1/release/create
              │
              ▼
      ┌───────────────┐
      │ 1. 参数校验    │
      │  - 必填字段    │
      │  - 格式合法性  │
      └───────┬───────┘
              ▼
      ┌───────────────┐
      │ 2. 兼容性检查  │
      │  - 正式发布    │
      │    唯一性检查  │
      │  - 多任务冲突  │
      │    检测        │
      └───────┬───────┘
              ▼
      ┌───────────────┐
      │ 3. 包合法性    │
      │  - 从 Kodo    │
      │    验证包存在  │
      │  - MD5 校验   │
      └───────┬───────┘
              ▼
      ┌───────────────┐
      │ 4. 灰度策略    │
      │    有效性验证  │
      │  - 节点是否    │
      │    被占用      │
      │  - 比例合法性  │
      └───────┬───────┘
              ▼
      ┌───────────────┐
      │ 5. 创建任务    │
      │  - 保存到      │
      │    MongoDB    │
      │  - 初始化灰度  │
      │    节点到Redis │
      │  - 记录历史    │
      └───────┬───────┘
              ▼
        [返回任务ID]
```

**关键逻辑**：

1. **唯一性约束**：同一应用、同一设备类型，正式发布任务同时只能有一个
2. **节点互斥**：节点不能同时参与多个正在进行的发布任务
3. **Main/Alter 配置**：
   - 新增组件：MainConfig 为空，AlterConfig 从请求获取
   - 升级组件：MainConfig 从 SysParam 表获取，AlterConfig 从请求获取

### 4.2 继续发布流程

```
[客户端] → POST /v1/release/:releaseID/continue
              │
              ▼
      ┌───────────────┐
      │ 1. 获取任务    │
      │  - 校验任务    │
      │    状态        │
      │  - 必须为      │
      │    processing │
      └───────┬───────┘
              ▼
      ┌───────────────┐
      │ 2. 比例调整    │
      │  - 计算新增    │
      │    节点数      │
      │  - 自动分配    │
      │    节点        │
      └───────┬───────┘
              ▼
      ┌───────────────┐
      │ 3. 手动增删    │
      │  - AddNodes   │
      │  - DelNodes   │
      │  - 冲突检测    │
      └───────┬───────┘
              ▼
      ┌───────────────┐
      │ 4. 更新灰度    │
      │  - Redis      │
      │  - MongoDB    │
      │  - 记录历史    │
      └───────┬───────┘
              ▼
        [返回成功]
```

**关键逻辑**：

- **增量更新**：只增加新的灰度节点，不影响已有节点
- **幂等性**：重复调用相同参数应产生相同结果
- **原子性**：节点更新要么全部成功，要么全部失败

### 4.3 完成发布流程

```
[客户端] → POST /v1/release/:releaseID/complete
              │
              ▼
      ┌───────────────┐
      │ 1. 校验任务    │
      │  - 状态检查    │
      │  - 权限检查    │
      └───────┬───────┘
              ▼
      ┌───────────────┐
      │ 2. 全量切换    │
      │  - 清空灰度    │
      │    节点列表    │
      │  - 更新Main    │
      │    配置        │
      └───────┬───────┘
              ▼
      ┌───────────────┐
      │ 3. 更新任务    │
      │  - 状态→      │
      │    completed  │
      │  - 设置EndAt  │
      │  - 记录历史    │
      └───────┬───────┘
              ▼
        [返回成功]
```

**关键逻辑**：

- **配置切换**：将 AlterConfig 提升为新的 MainConfig
- **清理缓存**：清空 Redis 中的灰度节点集合
- **不可逆性**：完成后不可再继续发布，只能回滚

### 4.4 回滚流程

```
[客户端] → POST /v1/release/:releaseID/rollback
              │
              ▼
      ┌───────────────┐
      │ 1. 校验任务    │
      │  - 是否可回滚  │
      │  - 仅正式发布  │
      │    可回滚      │
      └───────┬───────┘
              ▼
      ┌───────────────┐
      │ 2. 状态判断    │
      └───┬───────┬───┘
          │       │
    processing  completed
          │       │
          ▼       ▼
    ┌─────────┐ ┌─────────┐
    │清空灰度  │ │配置回退  │
    │节点      │ │Main→旧版│
    └─────────┘ └─────────┘
          │       │
          └───┬───┘
              ▼
      ┌───────────────┐
      │ 3. 保存灰度    │
      │    节点历史    │
      │  - MongoDB    │
      └───────┬───────┘
              ▼
      ┌───────────────┐
      │ 4. 更新任务    │
      │  - 状态→      │
      │    rollbacked │
      │  - 清空Redis  │
      │  - 记录历史    │
      └───────┬───────┘
              ▼
        [返回成功]
```

**关键逻辑**：

- **回滚限制**：
  - 只有正式发布可回滚
  - 进行中的任务：清空灰度节点即可
  - 已完成的任务：必须是最后一次正式发布才可回滚
- **配置恢复**：从历史记录中找到上一版本的 MainConfig
- **历史保存**：将当前灰度节点保存到 MongoDB，便于问题排查

### 4.5 灰度策略切换流程

```
[客户端] → POST /v1/release/:releaseID/filterswitch
              │
              ▼
      ┌───────────────┐
      │ 1. 模式选择    │
      └───┬───────┬───┘
          │       │
    ByNodeIds   ByFilter
          │       │
          ▼       ▼
    ┌─────────┐ ┌─────────┐
    │指定节点  │ │规则过滤  │
    │模式      │ │模式      │
    └─────────┘ └─────────┘
          │       │
          └───┬───┘
              ▼
      ┌───────────────┐
      │ 2. 清空当前    │
      │    灰度节点    │
      └───────┬───────┘
              ▼
      ┌───────────────┐
      │ 3. 应用新策略  │
      │  - 分配新节点  │
      │  - 更新Redis  │
      └───────┬───────┘
              ▼
        [返回成功]
```

## 5. 接口设计

### 5.1 创建发布任务

**请求**

```http
POST /v1/release/create
Content-Type: application/json

{
  "devType": "node",
  "appName": "example-app",
  "describe": "升级到 v2.0.0",
  "releaseType": "formal",
  "opType": "update",
  "appConfig": {
    "url": "https://kodo.example.com/packages/example-app-v2.0.0.tar.gz",
    "type": "tar.gz",
    "cmd": "example-app",
    "args": ["--config", "/etc/app/config.yaml"],
    "dir": "/opt/app",
    "md5": "5d41402abc4b2a76b9719d911017c592"
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

**响应**

```json
{
  "code": 0,
  "msg": "success"
}
```

### 5.2 继续发布

**请求**

```http
POST /v1/release/{releaseID}/continue
Content-Type: application/json

{
  "percentage": 30,
  "addNodes": ["node-001", "node-002"],
  "delNodes": []
}
```

**响应**

```json
{
  "code": 0,
  "msg": "success"
}
```

### 5.3 完成发布

**请求**

```http
POST /v1/release/{releaseID}/complete
```

**响应**

```json
{
  "code": 0,
  "msg": "success"
}
```

### 5.4 回滚发布

**请求**

```http
POST /v1/release/{releaseID}/rollback
```

**响应**

```json
{
  "code": 0,
  "msg": "success"
}
```

### 5.5 查询发布详情

**请求**

```http
GET /v1/release/{releaseID}/detail
```

**响应**

```json
{
  "id": "507f1f77bcf86cd799439011",
  "app": "example-app",
  "deviceType": "node",
  "releaseType": "formal",
  "opType": "update",
  "mainConfig": {
    "url": "https://kodo.example.com/packages/example-app-v1.0.0.tar.gz",
    "type": "tar.gz",
    "cmd": "example-app",
    "md5": "098f6bcd4621d373cade4e832627b4f6"
  },
  "alterConfig": {
    "url": "https://kodo.example.com/packages/example-app-v2.0.0.tar.gz",
    "type": "tar.gz",
    "cmd": "example-app",
    "md5": "5d41402abc4b2a76b9719d911017c592"
  },
  "grayPolicy": {
    "percentage": 30,
    "nodeIds": [],
    "filter": {
      "status": "online",
      "stages": ["prod"]
    },
    "statInfo": {
      "totalCount": 1000,
      "allowCount": 300
    }
  },
  "state": "processing",
  "rollbackAllowed": true,
  "operator": "admin",
  "desc": "升级到 v2.0.0",
  "createAt": 1698000000,
  "updateAt": 1698010000,
  "endAt": 0,
  "allowNodes": ["node-001", "node-002", "..."],
  "allowNodesTotal": 300
}
```

### 5.6 查询可用包列表

**请求**

```http
POST /v1/release/packages
Content-Type: application/json

{
  "devType": "node",
  "app": "example-app",
  "size": 10,
  "path": "releases/example-app/"
}
```

**响应**

```json
{
  "packages": [
    {
      "url": "https://kodo.example.com/releases/example-app/example-app-v2.0.0.tar.gz",
      "file": "example-app-v2.0.0.tar.gz",
      "md5": "5d41402abc4b2a76b9719d911017c592",
      "size": "10.5MB"
    },
    {
      "url": "https://kodo.example.com/releases/example-app/example-app-v1.9.0.tar.gz",
      "file": "example-app-v1.9.0.tar.gz",
      "md5": "098f6bcd4621d373cade4e832627b4f6",
      "size": "10.2MB"
    }
  ]
}
```

### 5.7 组件管理

#### 5.7.1 获取组件列表

**请求**

```http
GET /v1/release/allowapps?nodeType=node&app=example-app
```

**响应**

```json
{
  "apps": [
    {
      "id": "507f1f77bcf86cd799439011",
      "name": "example-app",
      "nodeType": "node",
      "path": "releases/example-app/",
      "operator": "admin",
      "createAt": 1698000000,
      "desc": "示例应用"
    }
  ]
}
```

#### 5.7.2 新增/更新/删除组件

**请求**

```http
POST /v1/release/allowapps
Content-Type: application/json

{
  "operation": "add",
  "name": "example-app",
  "nodeTypes": ["node", "smallBox"],
  "path": "releases/example-app/",
  "desc": "示例应用"
}
```

**响应**

```json
{
  "code": 0,
  "msg": "success"
}
```

### 5.8 节点搜索

**请求**

```http
POST /v1/nodes/search
Content-Type: application/json

{
  "devType": "node",
  "stage": "prod",
  "status": "online",  // 可选值: online, outline
  "customerIds": [100, 200],
  "size": 10
}
```

**响应**

```json
{
  "nodes": [
    {
      "nodeId": "node-001",
      "deviceType": "node",
      "stage": "prod",
      "status": "online",
      "customerIDs": [100]
    }
  ],
  "total": 1
}
```

### 5.9 发布任务列表

**请求**

```http
POST /v1/release/list
Content-Type: application/json

{
  "app": "example-app",
  "states": ["processing", "completed"],
  "page": 1,
  "size": 20
}
```

**响应**

```json
{
  "items": [
    {
      "id": "507f1f77bcf86cd799439011",
      "app": "example-app",
      "deviceType": "node",
      "releaseType": "formal",
      "state": "processing",
      "operator": "admin",
      "createAt": 1698000000
    }
  ],
  "total": 1
}
```

### 5.10 导出灰度节点

**请求**

```http
GET /v1/release/{releaseID}/allownodes/export
```

**响应**

```json
{
  "nodes": ["node-001", "node-002", "node-003", "..."]
}
```

### 5.11 查询发布历史

**请求**

```http
GET /v1/release/{releaseID}/history
```

**响应**

```json
{
  "items": [
    {
      "operation": "create",
      "operator": "admin",
      "opTime": 1698000000,
      "remark": "创建发布任务",
      "beforeState": "",
      "afterState": "processing",
      "grayPolicyInfo": {
        "afterPercentage": 10
      }
    },
    {
      "operation": "continue",
      "operator": "admin",
      "opTime": 1698010000,
      "remark": "继续发布",
      "beforeState": "processing",
      "afterState": "processing",
      "grayPolicyInfo": {
        "beforePercentage": 10,
        "afterPercentage": 30
      }
    }
  ]
}
```

## 6. 数据库设计

### 6.1 MongoDB 集合

#### 6.1.1 allowApps（组件管理表）

```javascript
{
  _id: "507f1f77bcf86cd799439011",
  name: "example-app",
  nodeType: "node",
  path: "releases/example-app/",
  desc: "示例应用",
  operator: "admin",
  createAt: ISODate("2023-10-24T10:00:00Z"),
  updateAt: ISODate("2023-10-24T10:00:00Z")
}
```

**索引**

```javascript
db.allowApps.createIndex({ name: 1, nodeType: 1 }, { unique: true })
db.allowApps.createIndex({ nodeType: 1 })
db.allowApps.createIndex({ createAt: -1 })
```

#### 6.1.2 nodeRelease（发布任务表）

```javascript
{
  _id: "507f1f77bcf86cd799439011",
  app: "example-app",
  deviceType: "node",
  releaseType: "formal",
  opType: "update",
  mainConfig: {
    url: "...",
    type: "tar.gz",
    cmd: "...",
    args: [],
    dir: "...",
    healthUrl: "...",
    md5: "..."
  },
  alterConfig: { /* 同上 */ },
  grayPolicy: {
    nodeIds: [],
    filter: {
      devType: "node",
      customerIds: [100, 200],
      status: "online",
      stages: ["prod"]
    },
    percentage: 30,
    statInfo: {
      markID: "",
      totalCount: 1000,
      allowCount: 300,
      totalCountUpdateAt: ISODate("..."),
      allowCountUpdateAt: ISODate("...")
    }
  },
  state: "processing",
  operator: "admin",
  desc: "升级到 v2.0.0",
  createAt: ISODate("..."),
  updateAt: ISODate("..."),
  endAt: ISODate("...")
}
```

**索引**

```javascript
db.nodeRelease.createIndex({ app: 1, deviceType: 1, state: 1 })
db.nodeRelease.createIndex({ createAt: -1 })
db.nodeRelease.createIndex({ releaseType: 1, state: 1 })

// 唯一性约束索引(防止同一应用同设备类型并发正式发布)
db.nodeRelease.createIndex(
  { app: 1, deviceType: 1, releaseType: 1, state: 1 },
  { 
    partialFilterExpression: { 
      state: "processing", 
      releaseType: "formal" 
    }
  }
)
```

#### 6.1.3 grayNodes（灰度节点表）

```javascript
{
  _id: "507f1f77bcf86cd799439012",
  releaseId: "507f1f77bcf86cd799439011",
  nodeId: "node-001",
  createAt: ISODate("...")
}
```

**索引**

```javascript
db.grayNodes.createIndex({ releaseId: 1, nodeId: 1 }, { unique: true })
db.grayNodes.createIndex({ nodeId: 1 })
```

#### 6.1.4 nodeReleaseHistory（发布历史表）

```javascript
{
  _id: "507f1f77bcf86cd799439013",
  releaseId: "507f1f77bcf86cd799439011",
  operation: "continue",
  operator: "admin",
  opTime: ISODate("..."),
  beforeState: "processing",
  afterState: "processing",
  grayPolicyInfo: {
    nodeIdsAdd: ["node-100"],
    nodeIdsDel: [],
    afterFilter: { /* ... */ },
    filterChangeMode: "switch",
    beforePercentage: 10,
    afterPercentage: 30
  },
  appConfigInfo: {
    beforeMain: { /* ... */ },
    afterMain: { /* ... */ }
  },
  createAt: ISODate("..."),
  updateAt: ISODate("...")
}
```

**索引**

```javascript
db.nodeReleaseHistory.createIndex({ releaseId: 1, opTime: -1 })
```

### 6.2 Redis 数据结构

#### 6.2.1 灰度节点集合

```
Key: jarvis_upd_config_allow_nodes_example-app_507f1f77bcf86cd799439011
Type: Set
Members: ["node-001", "node-002", "node-003", ...]
TTL: 604800秒(7天) - 作为安全网防止内存泄漏

注: 
- 活跃发布任务应定期刷新TTL
- 完成/回滚的发布任务应主动删除对应key
- 保守的7天TTL可防止系统异常时的内存泄漏
```

**操作**

```bash
# 添加节点
SADD jarvis_upd_config_allow_nodes_example-app_507f1f77bcf86cd799439011 node-001 node-002

# 移除节点
SREM jarvis_upd_config_allow_nodes_example-app_507f1f77bcf86cd799439011 node-001

# 查询节点数量
SCARD jarvis_upd_config_allow_nodes_example-app_507f1f77bcf86cd799439011

# 判断节点是否存在
SISMEMBER jarvis_upd_config_allow_nodes_example-app_507f1f77bcf86cd799439011 node-001

# 获取所有节点(大规模场景推荐使用SSCAN分页)
SSCAN jarvis_upd_config_allow_nodes_example-app_507f1f77bcf86cd799439011 0 COUNT 100

# 清空节点
DEL allowNodes:node:example-app:507f1f77bcf86cd799439011
```

## 7. 多版本并行发布设计

### 7.1 设计思路

**问题**：如何实现 v1.1 还未全量发布完成时，就可以启动 v1.2 的发布？

**解决方案**：

1. **任务隔离**：每个发布任务都有独立的 ReleaseID 和灰度节点列表
2. **节点互斥**：通过节点冲突检测，确保同一节点不会同时参与多个任务
3. **配置分层**：MainConfig 代表线上版本，AlterConfig 代表灰度版本

### 7.2 场景示例

**场景**：当前线上版本是 v1.0，要发布 v1.1 和 v1.2

**步骤 1：创建 v1.1 发布任务**

```http
POST /v1/release/create
{
  "appName": "example-app",
  "opType": "update",
  "appConfig": { "url": "v1.1.tar.gz", ... },
  "grayPolicy": { "percentage": 20 }
}
```

- 任务状态：processing
- MainConfig：v1.0
- AlterConfig：v1.1
- 灰度节点：200 个（假设共 1000 个节点）

**步骤 2：创建 v1.2 发布任务**

```http
POST /v1/release/create
{
  "appName": "example-app",
  "opType": "update",
  "appConfig": { "url": "v1.2.tar.gz", ... },
  "grayPolicy": { "percentage": 10 }
}
```

- 任务状态：processing
- MainConfig：v1.0（从 SysParam 读取，此时仍是 v1.0）
- AlterConfig：v1.2
- 灰度节点：80 个（从剩余 800 个节点中选择，不与 v1.1 冲突）

**步骤 3：完成 v1.1 发布**

```http
POST /v1/release/{v1.1-releaseID}/complete
```

- v1.1 任务状态：completed
- SysParam 中的 MainConfig 更新为 v1.1
- v1.1 的灰度节点清空

**步骤 4：继续 v1.2 发布**

```http
POST /v1/release/{v1.2-releaseID}/continue
{
  "percentage": 30
}
```

- v1.2 任务状态：仍为 processing
- MainConfig：v1.1（从 SysParam 读取，已更新）
- AlterConfig：v1.2
- 灰度节点：300 个（可包含之前 v1.1 灰度过的节点）

**结果**：

- v1.1 和 v1.2 可以并行进行，互不干扰
- 节点资源通过互斥机制合理分配
- MainConfig 始终反映线上最新稳定版本

### 7.3 冲突处理

**节点冲突检测逻辑**：

```go
func EnsureNodesNotInOtherTasks(
    ctx context.Context,
    release *NodeRelease,
    processingTasks []*NodeRelease,
    nodesToAdd []string,
) ([]string, error) {
    var tips []string
    
    for _, task := range processingTasks {
        if task.ID == release.ID {
            continue // 跳过当前任务
        }
        
        // 从 Redis 获取该任务的灰度节点
        taskNodes := redis.SMembers(getTopicKey(task))
        
        // 检查交集
        for _, nodeId := range nodesToAdd {
            if taskNodes.Contains(nodeId) {
                tips = append(tips, fmt.Sprintf(
                    "节点 %s 已在任务 %s 中",
                    nodeId, task.ID,
                ))
            }
        }
    }
    
    if len(tips) > 0 {
        return tips, errors.New("存在节点冲突")
    }
    
    return nil, nil
}
```

## 8. 技术选型

### 8.1 后端框架

**Go-Zero**

- 高性能：内置负载均衡、限流、熔断
- 工具链完善：goctl 自动生成代码
- 微服务友好：天然支持服务拆分

### 8.2 数据存储

**MongoDB**

- 文档模型：适合灵活的发布任务数据结构
- 高性能：支持高并发读写
- 索引丰富：支持复杂查询

**Redis**

- 高速缓存：毫秒级响应
- 数据结构丰富：Set 结构天然适合节点集合
- 原子操作：保证并发安全

### 8.3 对象存储

**Kodo（七牛云存储）**

- 高可用：多地域容灾
- CDN 加速：全球节点就近下载
- SDK 完善：Go SDK 开箱即用

### 8.4 依赖库

```go
require (
    github.com/qiniu/go-sdk/v7 v7.11.1        // Kodo SDK
    github.com/zeromicro/go-zero v1.6.0       // Go-Zero 框架
    go.mongodb.org/mongo-driver v1.12.1       // MongoDB 驱动
    github.com/go-redis/redis/v8 v8.11.5      // Redis 驱动（间接依赖）
)
```

## 9. 部署方案

### 9.1 单机部署

```
┌────────────────────────────────────┐
│         Load Balancer (Nginx)      │
└────────────┬───────────────────────┘
             │
    ┌────────┴────────┐
    │                 │
┌───┴────┐      ┌────┴────┐
│ App    │      │  App    │
│Instance│      │Instance │
│  :8080 │      │  :8081  │
└───┬────┘      └────┬────┘
    │                │
    └────────┬───────┘
             │
    ┌────────┴────────┐
    │                 │
┌───┴────┐      ┌────┴────┐
│MongoDB │      │  Redis  │
│ :27017 │      │  :6379  │
└────────┘      └─────────┘
```

### 9.2 高可用部署

```
┌────────────────────────────────────────────┐
│           Global Load Balancer             │
└────────────┬───────────────────────────────┘
             │
    ┌────────┴────────┐
    │                 │
┌───┴────────┐  ┌────┴────────┐
│ Zone A     │  │  Zone B     │
│ ┌────────┐ │  │  ┌────────┐ │
│ │App*3   │ │  │  │App*3   │ │
│ └────┬───┘ │  │  └────┬───┘ │
│      │     │  │       │     │
│ ┌────┴───┐ │  │  ┌────┴───┐ │
│ │MongoDB │ │  │  │MongoDB │ │
│ │Replica │ │  │  │Replica │ │
│ │  Set   │ │  │  │  Set   │ │
│ └────┬───┘ │  │  └────┬───┘ │
│      │     │  │       │     │
│ ┌────┴───┐ │  │  ┌────┴───┐ │
│ │ Redis  │ │  │  │ Redis  │ │
│ │Sentinel│ │  │  │Sentinel│ │
│ └────────┘ │  │  └────────┘ │
└────────────┘  └─────────────┘
```

## 10. 监控与运维

### 10.1 监控指标

**系统指标**

- QPS（每秒请求数）
- 响应时间（P50/P95/P99）
- 错误率

**业务指标**

- 发布任务数（processing/completed/rollbacked）
- 灰度节点数
- 发布成功率
- 回滚率

**资源指标**

- CPU 使用率
- 内存使用率
- 数据库连接数
- Redis 连接数

### 10.2 日志规范

**日志级别**

- ERROR：系统错误、操作失败
- WARN：异常情况但不影响主流程
- INFO：关键业务操作（创建、完成、回滚）
- DEBUG：调试信息

**日志格式**

```json
{
  "level": "INFO",
  "time": "2023-10-24T10:00:00Z",
  "module": "noderelease",
  "action": "create",
  "releaseId": "507f1f77bcf86cd799439011",
  "app": "example-app",
  "operator": "admin",
  "msg": "发布任务创建成功"
}
```

### 10.3 告警规则

| 指标 | 阈值 | 级别 | 动作 |
|------|------|------|------|
| 错误率 > 5% | 1分钟内 | P1 | 短信+电话 |
| 响应时间 > 1s | P95 超过 | P2 | 短信 |
| MongoDB 连接失败 | 任意次 | P0 | 电话 |
| Redis 连接失败 | 任意次 | P1 | 短信 |
| 回滚率 > 10% | 1小时内 | P2 | 短信 |

## 11. 安全设计

### 11.1 认证授权

- **RBAC 权限模型**：基于角色的访问控制
- **操作审计**：所有操作记录到 ReleaseHistory
- **敏感字段脱敏**：日志中不输出密码、密钥

### 11.2 数据安全

- **包 MD5 校验**：确保包完整性，防止篡改
- **HTTPS 传输**：API 调用强制 HTTPS
- **Redis 密码认证**：生产环境必须启用

## 12. 性能优化

### 12.1 查询优化

- **MongoDB 索引**：为常用查询字段建立索引
- **分页查询**：大数据量使用游标分页
- **Redis 缓存**：热点数据缓存到 Redis

### 12.2 并发优化

- **批量操作**：节点批量插入、批量删除
- **异步处理**：非关键路径异步化
- **连接池**：复用数据库连接

### 12.3 性能指标

| 操作 | 目标延迟 | 备注 |
|------|----------|------|
| 创建任务 | < 500ms | 包括数据库写入 |
| 继续发布 | < 300ms | 节点数 < 1000 |
| 查询详情 | < 100ms | 使用缓存 |
| 完成发布 | < 200ms | 包括配置更新 |
| 回滚 | < 500ms | 包括历史保存 |

## 13. 前端界面设计

### 13.1 发布任务管理界面

#### 13.1.1 发布任务列表 (release-list.html)

**功能特性**:
- 任务状态筛选(进行中/已完成/已回滚)
- 发布类型筛选(正式发布/测试发布)
- 关键词搜索(应用名/操作人)
- 分页展示
- 任务详情跳转

**页面元素**:
- 顶部导航:发布任务、组件管理、节点模拟器
- 搜索栏:状态、类型、关键词筛选
- 操作按钮:新建发布任务
- 任务列表:展示任务基本信息和操作入口

#### 13.1.2 创建发布任务 (release-create.html)

**表单字段**:
- 基本信息:设备类型、应用名、发布描述
- 发布类型:正式发布/功能验证
- 操作类型:新增/升级/移除组件
- 应用配置:包地址、启动命令、工作目录等
- 灰度策略:指定节点/规则过滤、灰度比例

**交互逻辑**:
- 选择应用后自动加载可用包列表
- 实时预览符合灰度条件的节点数量
- 表单验证和错误提示

#### 13.1.3 发布任务详情 (release-detail.html)

**信息展示**:
- 任务基本信息
- 当前版本配置和灰度版本配置
- 灰度策略和节点列表
- 操作历史时间线

**操作功能**:
- 继续发布(调整灰度比例/增删节点)
- 完成发布(全量切换)
- 回滚发布
- 导出灰度节点
- 查看操作历史

### 13.2 组件管理界面 (component-management.html)

**功能特性**:
- 组件列表展示
- 按名称、节点类型搜索
- 新增组件
- 编辑组件信息(路径、描述)
- 删除组件
- 分页展示

**表单字段**:
- 组件名称
- 节点类型(大节点/小盒子)
- Kodo 存放路径
- 组件描述

### 13.3 节点模拟器界面 (node-simulator.html)

**功能特性**:
- 手动输入节点 ID
- 随机生成测试节点
- 批量节点模拟
- 节点状态展示
- 启动/停止模拟

**使用场景**:
- 发布功能测试
- 灰度策略验证
- 系统演示

### 13.4 前端技术栈

- **原生 JavaScript**:无框架依赖,轻量级实现
- **CSS3**:现代化 UI 设计,响应式布局
- **Fetch API**:与后端 RESTful API 交互
- **本地代理**:simple-proxy.js 提供开发环境跨域支持

## 14. 工具和测试

### 14.1 节点数据生成工具

**位置**: `tools/generate_nodejoin_data/`

**功能**:
- 批量生成模拟节点数据
- 支持自定义节点数量、类型、业务 ID
- 用于开发和测试环境数据准备

**使用方法**:
```bash
cd tools/generate_nodejoin_data
go run main.go -count=1000 -devType=node -stage=prod
```

### 14.2 测试包目录

**位置**: `test/pkgs/`

**结构**:
```
test/pkgs/
└── app1/
    ├── 20251025-app1-tag-v1.0/
    │   └── app1
    ├── 20251025-app1-tag-v1.1/
    │   └── app1
    └── 20251025-app1-tag-v1.2/
        └── app1
```

**用途**:
- 提供测试用的应用版本包
- 模拟版本升级场景
- 验证发布和回滚功能

## 15. 项目进展

### 15.1 Phase 1（已完成）

- ✅ 灰度发布基础能力
- ✅ 手动回滚
- ✅ 多版本并行发布
- ✅ 节点级灰度控制
- ✅ 组件管理系统
- ✅ 节点搜索功能
- ✅ 完整的 Web 管理界面
- ✅ 节点模拟器测试工具

### 15.2 Phase 2（规划中）

- 🔲 自动健康检查
- 🔲 自动回滚
- 🔲 蓝绿发布
- 🔲 金丝雀发布增强

### 15.3 Phase 3（长期规划）

- 🔲 K8S 环境适配
- 🔲 多云部署支持
- 🔲 AI 预测发布风险
- 🔲 可视化发布大盘

---

## 附录

### A. 术语表

| 术语 | 英文 | 解释 |
|------|------|------|
| 灰度发布 | Canary Deployment | 渐进式发布，先小范围验证再全量 |
| 蓝绿发布 | Blue-Green Deployment | 维护两套环境，切换流量实现发布 |
| 滚动发布 | Rolling Update | 逐个实例更新 |
| 回滚 | Rollback | 恢复到上一个稳定版本 |
| 灰度节点 | Gray Node | 参与灰度发布的节点 |
| Main 配置 | Main Config | 当前线上运行的版本配置 |
| Alter 配置 | Alter Config | 灰度中的新版本配置 |

### B. 参考资料

- [Go-Zero 官方文档](https://go-zero.dev/)
- [MongoDB 文档](https://www.mongodb.com/docs/)
- [Redis 文档](https://redis.io/docs/)
- [七牛云 Kodo](https://developer.qiniu.com/kodo)

### C. 版本历史

| 版本 | 日期 | 作者 | 说明 |
|------|------|------|------|
| v1.0 | 2025-10-24 | Claude | 初始版本 |
| v1.1 | 2025-10-25 | Claude | 新增组件管理、节点搜索、节点模拟器、前端界面设计章节 |

---

**文档结束**
