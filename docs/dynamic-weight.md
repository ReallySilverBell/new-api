# 动态权重系统

## 概述

动态权重系统是一个智能的渠道选择优化机制，通过动态调整渠道权重来提高系统的稳定性和可用性。当渠道出现临时性问题（如限流、服务器错误）时，系统会自动降低该渠道的选择概率，避免持续使用问题渠道；当渠道恢复正常后，权重会逐步恢复。

## 核心特性

### 1. 双层权重系统

```
最终有效权重 = 静态权重(Weight) × 动态权重因子(DynamicFactor)
```

- **静态权重(Weight)**：管理员配置的基础权重，不变
- **动态权重因子(DynamicFactor)**：根据运行状态动态调整，范围 [0.0039, 1.0]

### 2. 失败惩罚算法（指数级加速衰减）

当渠道出现应该惩罚的错误时，动态权重因子会快速下降：

| 连续失败次数 | 动态因子计算 | 动态因子值 | 有效权重（假设静态=100） |
|------------|------------|-----------|---------------------|
| 0次（正常） | 1.0 | 100% | 100 |
| 1次 | 0.5^1 | 50% | 50 |
| 2次 | 0.5^2 | 25% | 25 |
| 3次 | 0.5^4 | 6.25% | 6.25 |
| 4次及以上 | 0.5^8 | 0.39% | 0.39 |

**特点**：
- 第1-2次失败：温和降权，给渠道恢复机会
- 第3次失败：大幅降权，避免持续使用问题渠道
- 第4次及以上：保持最低权重，但不完全禁用（保留0.39%的概率）

### 3. 混合恢复算法

#### 渐进式恢复（成功触发）

每次请求成功时，立即恢复20%的权重：

```
恢复路径（从0.39%恢复到100%）：
第1次成功：0.39% + 20% = 20.39%
第2次成功：20.39% + 20% = 40.39%
第3次成功：40.39% + 20% = 60.39%
第4次成功：60.39% + 20% = 80.39%
第5次成功：80.39% + 20% = 100%（完全恢复）
```

#### 时间衰减恢复（后台自动）

失败后进入1小时冷却期，冷却结束后每小时自动恢复10%：

```
时间线（从0.39%开始，无成功请求）：
失败时刻：0.39%，开始1小时冷却
1小时后：0.39%（冷却结束，开始恢复）
2小时后：0.39% + 10% = 10.39%
3小时后：10.39% + 10% = 20.39%
...
11小时后：90.39% + 10% = 100%（完全恢复）
```

**注意**：每次失败都会重置冷却时间

### 4. 错误分类处理

#### 应该惩罚的错误（临时性问题）
- 429 Too Many Requests（限流）
- 500/502/503 服务器错误
- 超时错误（非504/524）
- 网络连接错误
- 响应时间过长

#### 应该直接禁用的错误（永久性问题）
- 401 Unauthorized（密钥无效）
- 403 Forbidden（权限不足）
- invalid_api_key
- account_deactivated
- billing_not_active
- insufficient_quota（余额不足）

#### 不应该惩罚的错误（客户端问题）
- 400 Bad Request（请求格式错误）
- 413 Request Entity Too Large（请求体过大）
- 客户端参数错误

## API接口

### 1. 获取所有渠道的动态权重状态

```http
GET /api/channel/dynamic-metrics
Authorization: Bearer <admin_token>
```

**响应示例**：
```json
{
  "enabled": true,
  "config": {
    "enabled": true,
    "recovery_step": 0.2,
    "cooldown_period": 3600,
    "time_recovery_rate": 0.1,
    "min_factor": 0.00390625,
    "max_consecutive_fails": 4
  },
  "channels": [
    {
      "channel_id": 1,
      "channel_name": "OpenAI-Main",
      "dynamic_factor": 1.0,
      "consecutive_fails": 0,
      "last_fail_time": 0,
      "last_success_time": 1707523200,
      "cooldown_end_time": 0,
      "total_requests": 1523,
      "failed_requests": 12,
      "success_rate": 0.9921,
      "effective_weight": 100,
      "static_weight": 100,
      "status": "正常"
    },
    {
      "channel_id": 2,
      "channel_name": "OpenAI-Backup",
      "dynamic_factor": 0.0625,
      "consecutive_fails": 3,
      "last_fail_time": 1707520000,
      "last_success_time": 1707519000,
      "cooldown_end_time": 1707523600,
      "total_requests": 856,
      "failed_requests": 45,
      "success_rate": 0.9474,
      "effective_weight": 6,
      "static_weight": 100,
      "status": "冷却中 (剩余45分钟)"
    }
  ],
  "updated_at": 1707523200
}
```

### 2. 获取动态权重配置

```http
GET /api/channel/dynamic-config
Authorization: Bearer <admin_token>
```

### 3. 更新动态权重配置

```http
PUT /api/channel/dynamic-config
Authorization: Bearer <root_token>
Content-Type: application/json

{
  "enabled": true,
  "recovery_step": 0.2,
  "cooldown_period": 3600,
  "time_recovery_rate": 0.1,
  "min_factor": 0.00390625,
  "max_consecutive_fails": 4
}
```

### 4. 手动触发时间恢复（测试用）

```http
POST /api/channel/trigger-recovery
Authorization: Bearer <root_token>
```

## 配置参数说明

| 参数 | 类型 | 默认值 | 说明 |
|-----|------|-------|------|
| enabled | bool | true | 是否启用动态权重系统 |
| recovery_step | float64 | 0.2 | 成功恢复步长（20%） |
| cooldown_period | int64 | 3600 | 冷却期（秒），默认1小时 |
| time_recovery_rate | float64 | 0.1 | 时间恢复速率（10%/小时） |
| min_factor | float64 | 0.00390625 | 最小动态因子（0.39%） |
| max_consecutive_fails | int | 4 | 最大连续失败次数 |

## 状态说明

| 状态 | 说明 |
|-----|------|
| 正常 | 动态因子 = 1.0，渠道完全可用 |
| 冷却中 | 失败后1小时内，暂停自动恢复 |
| 降权中 | 0.5 ≤ 动态因子 < 1.0 |
| 严重降权 | 动态因子 < 0.1 |
| 恢复中 | 冷却期结束后，正在自动恢复 |

## 使用场景

### 场景1：渠道限流

1. 渠道返回429错误
2. 动态因子降至50%（第1次失败）
3. 继续使用该渠道概率降低
4. 其他渠道承担更多流量
5. 1小时后或成功请求后逐步恢复

### 场景2：服务器临时故障

1. 渠道返回500/502/503错误
2. 连续失败3次，动态因子降至6.25%
3. 几乎不再使用该渠道
4. 故障恢复后，通过成功请求快速恢复（5次成功即可完全恢复）

### 场景3：网络波动

1. 渠道超时或连接错误
2. 动态因子逐步降低
3. 网络恢复后，权重自动恢复
4. 无需人工干预

## 优势

1. **快速响应**：失败后立即降低权重，避免继续使用问题渠道
2. **渐进恢复**：不会永久惩罚，给渠道恢复的机会
3. **保持可用性**：最低保留0.39%权重，避免完全饿死
4. **兼容现有架构**：不破坏优先级机制，只在同优先级内调整
5. **自动化**：无需人工干预，系统自动调整
6. **可配置**：所有参数都可调整，适应不同场景

## 监控建议

1. 定期查看 `/api/channel/dynamic-metrics` 了解渠道状态
2. 关注 `consecutive_fails` 较高的渠道
3. 监控 `success_rate` 低于90%的渠道
4. 对长期处于"严重降权"状态的渠道进行排查

## 注意事项

1. 动态权重系统与自动禁用功能独立运行
2. 永久性错误（如密钥无效）仍会触发自动禁用
3. 动态权重只影响同优先级内的渠道选择
4. 系统重启后动态权重状态会重置
5. 建议在生产环境启用，测试环境可以禁用

## 故障排查

### 问题1：渠道权重一直很低

**可能原因**：
- 渠道持续出现错误
- 冷却期未结束
- 没有成功请求触发恢复

**解决方案**：
- 检查渠道配置和密钥
- 等待冷却期结束
- 手动触发时间恢复

### 问题2：动态权重不生效

**可能原因**：
- 动态权重系统未启用
- 内存缓存未启用

**解决方案**：
- 检查配置 `enabled: true`
- 确保 `MEMORY_CACHE_ENABLED=true`

### 问题3：恢复速度太慢

**可能原因**：
- 成功请求太少
- 时间恢复速率太低

**解决方案**：
- 增加 `recovery_step`（如0.3）
- 增加 `time_recovery_rate`（如0.2）
- 减少 `cooldown_period`（如1800秒）
