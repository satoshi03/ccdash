# パフォーマンス最適化計画

## 優先度: 中（1ヶ月以内対応）

## 1. 現状の問題分析

### 主要なパフォーマンス問題
1. **N+1クエリ問題**
   - セッション取得時の関連データ個別取得
   - メッセージ取得時の非効率なクエリ

2. **非効率なゴルーチン管理**
   - パニックリカバリの不完全な実装
   - ワーカープールの基本的な管理

3. **大量データ処理の最適化不足**
   - バッチ処理の未実装
   - ページネーションの不完全な実装
   - メモリ効率の悪いデータ処理

## 2. パフォーマンス目標

### SLO（Service Level Objectives）
- API応答時間: P50 < 50ms, P99 < 200ms
- スループット: 1000 req/sec
- メモリ使用量: < 500MB（通常時）
- CPU使用率: < 30%（通常時）
- データベースクエリ: P99 < 100ms

## 3. 実装計画

### Phase 1: データベースクエリ最適化（1週間）

#### タスク1: N+1クエリの解決
```sql
-- Before: N+1 query
SELECT * FROM sessions WHERE project_id = ?;
-- For each session:
SELECT * FROM messages WHERE session_id = ?;
SELECT * FROM session_windows WHERE session_id = ?;

-- After: JOIN with batch loading
SELECT 
    s.*,
    m.id as message_id, m.content, m.timestamp,
    sw.id as window_id, sw.start_time, sw.end_time
FROM sessions s
LEFT JOIN messages m ON s.id = m.session_id
LEFT JOIN session_windows sw ON s.id = sw.session_id
WHERE s.project_id = ?
ORDER BY s.created_at DESC, m.timestamp;
```

- [ ] クエリ分析ツールの導入（EXPLAIN ANALYZE）
- [ ] 問題のあるクエリの特定
- [ ] JOINまたはバッチローディングへの変更
- [ ] プリロード戦略の実装

#### タスク2: インデックス最適化
```sql
-- 頻繁に使用されるクエリパターンに基づくインデックス
CREATE INDEX idx_sessions_project_created ON sessions(project_id, created_at DESC);
CREATE INDEX idx_messages_session_timestamp ON messages(session_id, timestamp);
CREATE INDEX idx_session_windows_session ON session_windows(session_id);
CREATE INDEX idx_jobs_status_created ON jobs(status, created_at DESC);
```

- [ ] クエリパターンの分析
- [ ] 必要なインデックスの特定
- [ ] インデックス作成とテスト
- [ ] 不要なインデックスの削除

#### タスク3: クエリキャッシュの実装
```go
// backend/internal/cache/query_cache.go
type QueryCache struct {
    cache *ristretto.Cache
    ttl   time.Duration
}

func (c *QueryCache) GetOrSet(key string, loader func() (interface{}, error)) (interface{}, error) {
    if val, found := c.cache.Get(key); found {
        return val, nil
    }
    
    val, err := loader()
    if err != nil {
        return nil, err
    }
    
    c.cache.SetWithTTL(key, val, 1, c.ttl)
    return val, nil
}
```

- [ ] キャッシュライブラリの選定（ristretto, bigcache）
- [ ] キャッシュ戦略の設計
- [ ] キャッシュ無効化ロジック
- [ ] キャッシュヒット率の監視

### Phase 2: ゴルーチン管理の改善（3日）

#### タスク1: ワーカープールの高度化
```go
// backend/internal/worker/pool.go
type WorkerPool struct {
    workers   int
    taskQueue chan Task
    results   chan Result
    errors    chan error
    wg        sync.WaitGroup
    ctx       context.Context
    cancel    context.CancelFunc
    metrics   *PoolMetrics
}

func (p *WorkerPool) Start() {
    for i := 0; i < p.workers; i++ {
        p.wg.Add(1)
        go p.worker(i)
    }
}

func (p *WorkerPool) worker(id int) {
    defer p.wg.Done()
    defer p.recoverPanic(id)
    
    for {
        select {
        case task := <-p.taskQueue:
            p.processTask(task)
        case <-p.ctx.Done():
            return
        }
    }
}
```

- [ ] 動的ワーカー数調整
- [ ] バックプレッシャー制御
- [ ] タスク優先度管理
- [ ] デッドレター キュー

#### タスク2: コンテキスト管理の改善
- [ ] タイムアウト付きコンテキスト
- [ ] キャンセレーション伝播
- [ ] コンテキスト値の適切な使用
- [ ] リクエストトレーシング

#### タスク3: パニックリカバリの完全実装
```go
func recoverMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        defer func() {
            if err := recover(); err != nil {
                // スタックトレースの記録
                stack := make([]byte, 4096)
                n := runtime.Stack(stack, false)
                
                log.Error("Panic recovered",
                    "error", err,
                    "stack", string(stack[:n]),
                    "request_id", c.GetString("request_id"),
                )
                
                c.AbortWithStatusJSON(500, gin.H{
                    "error": "Internal server error",
                })
            }
        }()
        c.Next()
    }
}
```

- [ ] 全ゴルーチンでのパニックリカバリ
- [ ] パニック時のログ記録
- [ ] アラート通知
- [ ] 自動復旧メカニズム

### Phase 3: 大量データ処理の最適化（4日）

#### タスク1: バッチ処理の実装
```go
// backend/internal/batch/processor.go
type BatchProcessor struct {
    batchSize int
    interval  time.Duration
    processor func([]interface{}) error
}

func (b *BatchProcessor) Process(items []interface{}) error {
    for i := 0; i < len(items); i += b.batchSize {
        end := min(i+b.batchSize, len(items))
        batch := items[i:end]
        
        if err := b.processor(batch); err != nil {
            return fmt.Errorf("batch processing failed at %d-%d: %w", i, end, err)
        }
        
        // Rate limiting
        time.Sleep(b.interval)
    }
    return nil
}
```

- [ ] ログ同期のバッチ処理
- [ ] データ集計のバッチ処理
- [ ] バルクインサート/アップデート
- [ ] 進捗報告機能

#### タスク2: ストリーミング処理の実装
```go
// backend/internal/streaming/processor.go
func ProcessJSONLStream(reader io.Reader, handler func(map[string]interface{}) error) error {
    decoder := json.NewDecoder(reader)
    
    for {
        var record map[string]interface{}
        if err := decoder.Decode(&record); err == io.EOF {
            break
        } else if err != nil {
            return err
        }
        
        if err := handler(record); err != nil {
            return err
        }
    }
    
    return nil
}
```

- [ ] メモリ効率的なJSONL処理
- [ ] チャンクベースの処理
- [ ] バックプレッシャー対応
- [ ] エラーハンドリング

#### タスク3: ページネーションの改善
```go
// backend/internal/pagination/cursor.go
type CursorPagination struct {
    Cursor string
    Limit  int
    Order  string
}

func (p *CursorPagination) Apply(query *sql.DB) *sql.DB {
    if p.Cursor != "" {
        decoded, _ := base64.DecodeString(p.Cursor)
        query = query.Where("id > ?", string(decoded))
    }
    
    return query.Order(p.Order).Limit(p.Limit)
}
```

- [ ] カーソルベースページネーション
- [ ] 効率的なカウントクエリ
- [ ] プリフェッチ戦略
- [ ] レスポンスキャッシュ

### Phase 4: メモリ最適化（3日）

#### タスク1: メモリプロファイリング
```go
import _ "net/http/pprof"

func setupProfiling() {
    go func() {
        log.Println(http.ListenAndServe("localhost:6060", nil))
    }()
}
```

- [ ] pprofの導入
- [ ] メモリリークの検出
- [ ] ヒープ割り当ての最適化
- [ ] GCチューニング

#### タスク2: オブジェクトプールの活用
```go
var bufferPool = sync.Pool{
    New: func() interface{} {
        return new(bytes.Buffer)
    },
}

func processWithPool(data []byte) {
    buf := bufferPool.Get().(*bytes.Buffer)
    defer func() {
        buf.Reset()
        bufferPool.Put(buf)
    }()
    
    // Use buffer...
}
```

- [ ] バッファプール
- [ ] 接続プール
- [ ] オブジェクトプール
- [ ] プール効率の監視

## 4. ベンチマークとテスト

### ベンチマークテスト
```go
func BenchmarkSessionQuery(b *testing.B) {
    db := setupTestDB()
    defer db.Close()
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        sessions, _ := GetRecentSessions(db, 100)
        _ = sessions
    }
}
```

- [ ] 各エンドポイントのベンチマーク
- [ ] データベースクエリのベンチマーク
- [ ] メモリ使用量の測定
- [ ] 並行処理のベンチマーク

### 負荷テスト
```yaml
# k6-script.js
import http from 'k6/http';
import { check } from 'k6';

export let options = {
    stages: [
        { duration: '2m', target: 100 },
        { duration: '5m', target: 100 },
        { duration: '2m', target: 200 },
        { duration: '5m', target: 200 },
        { duration: '2m', target: 0 },
    ],
};

export default function() {
    let response = http.get('http://localhost:6060/api/sessions');
    check(response, {
        'status is 200': (r) => r.status === 200,
        'response time < 200ms': (r) => r.timings.duration < 200,
    });
}
```

- [ ] k6またはvegeta導入
- [ ] シナリオベーステスト
- [ ] スパイクテスト
- [ ] ソークテスト

## 5. 監視とアラート

### メトリクス収集
- [ ] Prometheusメトリクス
  - [ ] HTTPレスポンスタイム
  - [ ] データベースクエリ時間
  - [ ] ゴルーチン数
  - [ ] メモリ使用量
  - [ ] CPU使用率

### ダッシュボード
- [ ] Grafanaダッシュボード作成
- [ ] リアルタイムメトリクス表示
- [ ] アラート設定
- [ ] SLO/SLI追跡

## 6. 実装スケジュール

### Week 1
- データベースクエリ最適化
- インデックス作成

### Week 2
- ゴルーチン管理改善
- キャッシュ実装

### Week 3
- バッチ処理実装
- ストリーミング処理

### Week 4
- メモリ最適化
- ベンチマークと負荷テスト
- 最終調整

## 7. 成功基準

- [ ] API応答時間: P99 < 200ms達成
- [ ] メモリ使用量: 30%削減
- [ ] データベースクエリ: 50%高速化
- [ ] 同時処理能力: 2倍向上
- [ ] エラー率: < 0.1%

## 8. リスクと対策

### リスク1: 最適化による複雑性増加
- **対策**: シンプルさを保つ
- **対策**: 十分なドキュメント化

### リスク2: 過度な最適化
- **対策**: プロファイリングに基づく最適化
- **対策**: ROIの評価

### リスク3: 互換性の破壊
- **対策**: 段階的な移行
- **対策**: 包括的なテスト