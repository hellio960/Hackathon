# NodeJoin 测试数据生成工具

## 功能说明

这个工具用于向 MongoDB 的 `nodeJoin` 集合插入测试数据，满足以下要求：

1. **NodeId**: 使用 `uuid.NewString()` 随机生成
2. **DeviceType**: `ant.A`, `ant.B`, `ant.C`, `jarvis.A` 各有至少 100 条数据
3. **Stage**: `register`, `submitted`, `censored`, `uncensored`, `inservice` 各有至少 100 条数据
4. **NodeType**: 根据 `deviceType` 自动决定：
   - `jarvis.A` → `node`
   - `ant.A/B/C` → `smallBox`
5. **Status**: 全部为 `online`
6. **CustomerIDs**: 当填了 `customerIDs` 时，该记录的 `stage` 一定是 `inservice`
   - 业务 ID：10001(业务A)、10002(业务B)、10003(业务C)、10004(业务D)
   - 每个业务在各设备类型下都有至少 100 条数据

## 使用方法

### 1. 确保 MongoDB 已启动

```bash
# 确认 MongoDB 运行在 localhost:27017
mongosh
```

### 2. 运行工具

```bash
cd /Users/liaojianhua/Documents/work/qiniu/code/Hackathon
go run tools/generate_nodejoin_data/main.go
```

### 3. 清空数据（如需重新生成）

```bash
mongosh
use jarvis
db.nodeJoin.deleteMany({})
```

## 生成的数据量

- **按阶段**: 5 个阶段 × 4 个设备类型 × 100-120 条 ≈ 2000-2400 条
- **按业务**: 4 个业务 × 4 个设备类型 × 100-120 条 ≈ 1600-1920 条
- **总计**: 约 3600-4320 条记录

## 数据分布

工具会在插入完成后自动打印统计信息，包括：
- 总记录数
- 按设备类型统计
- 按阶段统计
- 按业务统计
- 按设备类型和业务交叉统计

