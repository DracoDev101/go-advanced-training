# Lab 17：PostgreSQL、事务与 Repository

本 Lab 不依赖外部 PostgreSQL，使用 in-memory repository 模拟必须由 DB 事务保证的不变量；README 给出对应 SQL。

## 运行

```bash
go test ./...
go test -race ./...
go test -bench=. -benchmem ./...
go vet ./...
```

## 核心实验

- `UnsafeClaim` 模拟错误的 SELECT 后 UPDATE，测试中能复现 owner 被覆盖。
- `ClaimAtomic` 模拟 `UPDATE ... WHERE status='pending' RETURNING id`，并发下只有一个 worker 成功。
- `RequeueStale` 模拟 worker crash 后 stale lock 恢复。
- `CompleteWithOutbox` 模拟同一事务内写 job 状态和 outbox event。

## 正确 SQL

```sql
UPDATE jobs
SET status='running', locked_by=$2, locked_at=now()
WHERE id=$1 AND status='pending'
RETURNING id;
```

批量 claim：

```sql
WITH picked AS (
  SELECT id FROM jobs
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

## 排查命令

```sql
SELECT pid, wait_event_type, wait_event, query FROM pg_stat_activity WHERE wait_event_type IS NOT NULL;
SELECT * FROM pg_locks WHERE NOT granted;
EXPLAIN (ANALYZE, BUFFERS) SELECT ...;
```

## 生产启发

Go race detector 不能发现数据库层面的业务竞态；claim job 这种不变量必须用条件更新、事务、唯一约束或锁来保证。
