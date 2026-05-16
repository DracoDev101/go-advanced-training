# PostgreSQL 事务与锁排查手册

## 1. 连接池不是越大越好

Go 服务常见问题是 pool exhaustion：请求卡在等连接，而 CPU 不高。

需要观测：

```text
db_pool_open_connections
db_pool_in_use
db_pool_wait_count
db_pool_wait_duration_seconds
query_duration_seconds{operation}
```

Go `database/sql`：

```go
db.SetMaxOpenConns(20)
db.SetMaxIdleConns(20)
db.SetConnMaxLifetime(time.Hour)
```

## 2. Job Claim 正确写法

错误模式：

```sql
SELECT status FROM jobs WHERE id=$1;
-- app 判断 pending
UPDATE jobs SET status='running' WHERE id=$1;
```

正确模式：

```sql
UPDATE jobs
SET status='running', locked_by=$2, locked_at=now()
WHERE id=$1 AND status='pending'
RETURNING id;
```

返回 0 行代表已经被其他 worker claim，不是错误。

## 3. SKIP LOCKED 批量领取

```sql
WITH picked AS (
  SELECT id
  FROM jobs
  WHERE status='pending'
  ORDER BY priority DESC, created_at ASC
  FOR UPDATE SKIP LOCKED
  LIMIT $1
)
UPDATE jobs
SET status='running', locked_by=$2, locked_at=now()
WHERE id IN (SELECT id FROM picked)
RETURNING *;
```

适合多 worker 抢任务，避免互相等待。

## 4. Lock Wait 排查

```sql
SELECT pid, wait_event_type, wait_event, query
FROM pg_stat_activity
WHERE wait_event_type IS NOT NULL;
```

```sql
SELECT * FROM pg_locks WHERE NOT granted;
```

## 5. Deadlock 处理

PostgreSQL 会主动 abort 一个事务。应用层应识别 retryable error，做有限重试：

```text
deadlock_detected
serialization_failure
```

重试必须满足：

- 事务函数可重入。
- 写操作幂等或有唯一约束。
- retry 有 backoff 和最大次数。

## 6. 事务边界

事务里不要做：

- HTTP/gRPC 下游调用。
- 长时间 CPU 计算。
- 等 MQ ack。
- 用户交互。

事务应该只包住必须一致的 DB 状态变化。

## 7. Outbox Pattern

解决 “DB 成功但事件发布失败”：

```text
同一事务：写业务表 + 写 outbox 表
后台 publisher：读取 outbox → 发布消息 → 标记 published
consumer：按 event_id 幂等处理
```
