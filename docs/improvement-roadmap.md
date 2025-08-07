# CCDash 改善ロードマップ

## 概要

このドキュメントは、コードレビューで特定された問題に対する具体的な改善計画と実装ガイドラインを提供します。

## フェーズ1: 緊急対応（1週間）

### 1.1 SQLインジェクション対策

#### 現状の問題
```go
// 危険な例
query := fmt.Sprintf("SELECT * FROM sessions WHERE id = '%s'", sessionID)
```

#### 改善案
```go
// 安全な実装
query := "SELECT * FROM sessions WHERE id = ?"
rows, err := db.Query(query, sessionID)
```

**実装タスク:**
- [ ] 全データベースクエリの監査
- [ ] パラメータ化クエリへの変換
- [ ] SQLビルダーライブラリ（squirrel等）の導入検討

### 1.2 API認証強化

#### 実装案
```go
// backend/internal/middleware/auth_improved.go
type AuthMiddleware struct {
    tokenStore TokenStore
    rateLimiter RateLimiter
}

func (m *AuthMiddleware) Authenticate() gin.HandlerFunc {
    return func(c *gin.Context) {
        token := c.GetHeader("X-API-Key")
        if token == "" {
            token = c.GetHeader("Authorization")
        }
        
        // トークン検証
        if !m.tokenStore.Validate(token) {
            c.AbortWithStatusJSON(401, gin.H{"error": "Unauthorized"})
            return
        }
        
        // レート制限
        if !m.rateLimiter.Allow(token) {
            c.AbortWithStatusJSON(429, gin.H{"error": "Rate limit exceeded"})
            return
        }
        
        c.Next()
    }
}
```

### 1.3 データベース接続プール

#### 実装案
```go
// backend/internal/database/pool.go
type DBPool struct {
    db *sql.DB
    config PoolConfig
}

type PoolConfig struct {
    MaxOpenConns    int
    MaxIdleConns    int
    ConnMaxLifetime time.Duration
}

func NewDBPool(cfg PoolConfig) (*DBPool, error) {
    db, err := sql.Open("duckdb", cfg.DatabasePath)
    if err != nil {
        return nil, err
    }
    
    db.SetMaxOpenConns(cfg.MaxOpenConns)
    db.SetMaxIdleConns(cfg.MaxIdleConns)
    db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
    
    return &DBPool{db: db, config: cfg}, nil
}
```

## フェーズ2: 短期改善（1ヶ月）

### 2.1 統一エラーハンドリング

#### エラー型定義
```go
// backend/internal/errors/errors.go
type AppError struct {
    Code    string
    Message string
    Details interface{}
    Err     error
}

const (
    ErrCodeValidation   = "VALIDATION_ERROR"
    ErrCodeNotFound     = "NOT_FOUND"
    ErrCodeUnauthorized = "UNAUTHORIZED"
    ErrCodeInternal     = "INTERNAL_ERROR"
)

func NewValidationError(msg string, details interface{}) *AppError {
    return &AppError{
        Code:    ErrCodeValidation,
        Message: msg,
        Details: details,
    }
}
```

#### エラーミドルウェア
```go
// backend/internal/middleware/error.go
func ErrorHandler() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Next()
        
        if len(c.Errors) > 0 {
            err := c.Errors.Last()
            
            var appErr *AppError
            if errors.As(err, &appErr) {
                c.JSON(getStatusCode(appErr.Code), gin.H{
                    "error": appErr.Message,
                    "code":  appErr.Code,
                })
            } else {
                c.JSON(500, gin.H{
                    "error": "Internal server error",
                    "code":  ErrCodeInternal,
                })
            }
        }
    }
}
```

### 2.2 CORS設定改善

```go
// backend/internal/config/cors.go
func GetCORSConfig(env string) cors.Config {
    config := cors.DefaultConfig()
    
    switch env {
    case "production":
        config.AllowOrigins = []string{
            "https://ccdash.example.com",
        }
        config.AllowCredentials = true
        config.MaxAge = 12 * time.Hour
        
    case "development":
        config.AllowOrigins = []string{
            "http://localhost:3000",
            "http://localhost:3001",
        }
        config.AllowCredentials = true
        
    default:
        // テスト環境
        config.AllowAllOrigins = true
    }
    
    config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
    config.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization", "X-API-Key"}
    
    return config
}
```

### 2.3 ゴルーチン管理改善

```go
// backend/internal/services/worker_pool.go
type WorkerPool struct {
    workers   int
    jobs      chan Job
    results   chan Result
    ctx       context.Context
    cancel    context.CancelFunc
    wg        sync.WaitGroup
}

func NewWorkerPool(workers int) *WorkerPool {
    ctx, cancel := context.WithCancel(context.Background())
    return &WorkerPool{
        workers: workers,
        jobs:    make(chan Job, workers*2),
        results: make(chan Result, workers*2),
        ctx:     ctx,
        cancel:  cancel,
    }
}

func (p *WorkerPool) Start() {
    for i := 0; i < p.workers; i++ {
        p.wg.Add(1)
        go p.worker(i)
    }
}

func (p *WorkerPool) worker(id int) {
    defer p.wg.Done()
    defer func() {
        if r := recover(); r != nil {
            log.Printf("Worker %d panic: %v", id, r)
        }
    }()
    
    for {
        select {
        case <-p.ctx.Done():
            return
        case job, ok := <-p.jobs:
            if !ok {
                return
            }
            result := p.processJob(job)
            p.results <- result
        }
    }
}

func (p *WorkerPool) Shutdown() {
    p.cancel()
    close(p.jobs)
    p.wg.Wait()
    close(p.results)
}
```

## フェーズ3: 中長期改善（3ヶ月）

### 3.1 レイヤードアーキテクチャ

```
/backend
├── cmd/                    # エントリーポイント
├── internal/
│   ├── domain/            # ドメインモデル・ビジネスルール
│   │   ├── models/
│   │   └── services/
│   ├── application/       # アプリケーションサービス
│   │   └── usecases/
│   ├── infrastructure/    # インフラ層
│   │   ├── database/
│   │   ├── cache/
│   │   └── external/
│   └── interfaces/        # インターフェース層
│       ├── api/
│       └── cli/
```

### 3.2 依存性注入

```go
// backend/internal/container/container.go
type Container struct {
    db             *sql.DB
    tokenService   *services.TokenService
    sessionService *services.SessionService
    // ... 他のサービス
}

func NewContainer(cfg *config.Config) (*Container, error) {
    db, err := database.NewDBPool(cfg.DatabaseConfig)
    if err != nil {
        return nil, err
    }
    
    return &Container{
        db:             db,
        tokenService:   services.NewTokenService(db),
        sessionService: services.NewSessionService(db),
        // ... 初期化
    }, nil
}
```

### 3.3 キャッシュ層の導入

```go
// backend/internal/cache/cache.go
type Cache interface {
    Get(key string) (interface{}, error)
    Set(key string, value interface{}, ttl time.Duration) error
    Delete(key string) error
}

type RedisCache struct {
    client *redis.Client
}

func (c *RedisCache) Get(key string) (interface{}, error) {
    val, err := c.client.Get(context.Background(), key).Result()
    if err == redis.Nil {
        return nil, ErrCacheMiss
    }
    return val, err
}
```

## テスト戦略

### ユニットテスト例
```go
func TestTokenService_CalculateCost(t *testing.T) {
    tests := []struct {
        name     string
        input    TokenUsage
        expected float64
    }{
        {
            name: "standard calculation",
            input: TokenUsage{
                InputTokens:  1000,
                OutputTokens: 500,
            },
            expected: 0.015,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            service := NewTokenService(mockDB)
            result := service.CalculateCost(tt.input)
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

### 統合テスト例
```go
func TestAPI_CreateSession(t *testing.T) {
    // Setup
    db := setupTestDB(t)
    defer db.Close()
    
    router := setupRouter(db)
    
    // Test
    w := httptest.NewRecorder()
    body := `{"project_name": "test"}`
    req := httptest.NewRequest("POST", "/api/sessions", strings.NewReader(body))
    req.Header.Set("X-API-Key", "test-key")
    
    router.ServeHTTP(w, req)
    
    // Assert
    assert.Equal(t, 201, w.Code)
    
    var response SessionResponse
    json.Unmarshal(w.Body.Bytes(), &response)
    assert.NotEmpty(t, response.ID)
}
```

## パフォーマンス最適化

### クエリ最適化
```sql
-- インデックス追加
CREATE INDEX idx_messages_session_id ON messages(session_id);
CREATE INDEX idx_messages_timestamp ON messages(timestamp);
CREATE INDEX idx_sessions_project_name ON sessions(project_name);

-- 複合インデックス
CREATE INDEX idx_session_windows_active_time 
ON session_windows(is_active, window_start, window_end);
```

### バッチ処理
```go
func BatchInsertMessages(db *sql.DB, messages []Message) error {
    tx, err := db.Begin()
    if err != nil {
        return err
    }
    defer tx.Rollback()
    
    stmt, err := tx.Prepare(`
        INSERT INTO messages (id, session_id, content, timestamp)
        VALUES (?, ?, ?, ?)
    `)
    if err != nil {
        return err
    }
    defer stmt.Close()
    
    for _, msg := range messages {
        _, err := stmt.Exec(msg.ID, msg.SessionID, msg.Content, msg.Timestamp)
        if err != nil {
            return err
        }
    }
    
    return tx.Commit()
}
```

## モニタリング

### メトリクス収集
```go
// backend/internal/metrics/metrics.go
var (
    requestDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "http_request_duration_seconds",
            Help: "Duration of HTTP requests in seconds",
        },
        []string{"method", "endpoint", "status"},
    )
    
    dbQueryDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "db_query_duration_seconds",
            Help: "Duration of database queries in seconds",
        },
        []string{"query_type"},
    )
)
```

## セキュリティチェックリスト

- [ ] 全入力値のバリデーション
- [ ] SQLインジェクション対策
- [ ] XSS対策
- [ ] CSRF対策
- [ ] レート制限
- [ ] 適切なHTTPSヘッダー
- [ ] シークレット管理
- [ ] 監査ログ
- [ ] 依存関係の脆弱性スキャン

## まとめ

この改善ロードマップに従うことで、CCDashプロジェクトの品質、セキュリティ、パフォーマンスを段階的に向上させることができます。各フェーズは前のフェーズの完了を前提としており、着実な改善を実現します。

---

*作成日: 2025-08-06*
*最終更新: 2025-08-06*