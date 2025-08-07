# アーキテクチャ改善計画

## 優先度: 中〜高（1ヶ月以内対応）

## 1. 現状の問題分析

### 主要な問題点
1. **グローバル状態の使用**
   - `GetGlobalInitializationService()`によるシングルトン
   - テスタビリティの低下
   - 並行処理での問題

2. **層の責務が不明確**
   - サービス層にデータアクセスとビジネスロジックが混在
   - ハンドラー層に複雑なビジネスロジック
   - モデル層の責務が曖昧

3. **エラーハンドリングの不統一**
   - エラー処理パターンの不統一
   - エラーメッセージの直接露出
   - エラーコードの欠如

## 2. 目標アーキテクチャ

### Clean Architecture原則の適用
```
┌─────────────────────────────────────────┐
│           Presentation Layer            │
│         (Handlers / Controllers)        │
├─────────────────────────────────────────┤
│           Application Layer             │
│           (Use Cases / DTOs)           │
├─────────────────────────────────────────┤
│            Domain Layer                 │
│     (Entities / Business Logic)        │
├─────────────────────────────────────────┤
│         Infrastructure Layer            │
│    (Database / External Services)      │
└─────────────────────────────────────────┘
```

## 3. 実装計画

### Phase 1: レイヤー分離（1週間）

#### タスク1: ドメイン層の確立
```go
// backend/internal/domain/session.go
type Session struct {
    ID          string
    ProjectID   string
    StartTime   time.Time
    EndTime     *time.Time
    TokenUsage  TokenUsage
}

// ビジネスルール
func (s *Session) CalculateDuration() time.Duration
func (s *Session) IsActive() bool
func (s *Session) CanMergeWith(other *Session) bool
```

- [ ] エンティティの定義
  - [ ] Session
  - [ ] Project
  - [ ] TokenUsage
  - [ ] SessionWindow
  - [ ] Job
- [ ] ビジネスルールの実装
- [ ] ドメインイベントの定義

#### タスク2: リポジトリインターフェースの定義
```go
// backend/internal/domain/repository/session_repository.go
type SessionRepository interface {
    FindByID(ctx context.Context, id string) (*domain.Session, error)
    Save(ctx context.Context, session *domain.Session) error
    FindByTimeRange(ctx context.Context, start, end time.Time) ([]*domain.Session, error)
}
```

- [ ] SessionRepository
- [ ] ProjectRepository
- [ ] JobRepository
- [ ] TokenUsageRepository

#### タスク3: アプリケーション層（ユースケース）の実装
```go
// backend/internal/application/usecase/session_usecase.go
type SessionUseCase struct {
    sessionRepo domain.SessionRepository
    projectRepo domain.ProjectRepository
    logger      Logger
}

func (u *SessionUseCase) GetRecentSessions(ctx context.Context, limit int) ([]*dto.SessionDTO, error)
```

- [ ] SessionUseCase
- [ ] ProjectUseCase
- [ ] JobUseCase
- [ ] SyncUseCase

### Phase 2: 依存性注入とDIコンテナ（3日）

#### タスク1: DIコンテナの選定と実装
- [ ] wire vs dig vs fx の評価
- [ ] DIコンテナの実装
- [ ] プロバイダー関数の作成

#### タスク2: グローバル状態の除去
```go
// backend/internal/container/wire.go
// +build wireinject

func InitializeApp(config *config.Config) (*Application, error) {
    wire.Build(
        database.NewConnection,
        repository.NewSessionRepository,
        usecase.NewSessionUseCase,
        handler.NewSessionHandler,
        NewApplication,
    )
    return nil, nil
}
```

- [ ] InitializationServiceのリファクタリング
- [ ] グローバル変数の除去
- [ ] 設定管理の改善

### Phase 3: エラーハンドリングの統一（3日）

#### タスク1: カスタムエラー型の定義
```go
// backend/internal/errors/errors.go
type AppError struct {
    Code       string
    Message    string
    StatusCode int
    Details    map[string]interface{}
    Cause      error
}

// エラーコード定義
const (
    ErrCodeNotFound     = "NOT_FOUND"
    ErrCodeValidation   = "VALIDATION_ERROR"
    ErrCodeUnauthorized = "UNAUTHORIZED"
    ErrCodeInternal     = "INTERNAL_ERROR"
)
```

- [ ] エラー型の定義
- [ ] エラーコードの体系化
- [ ] エラーファクトリー関数

#### タスク2: エラーミドルウェアの実装
```go
// backend/internal/middleware/error.go
func ErrorMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Next()
        
        if len(c.Errors) > 0 {
            err := c.Errors.Last()
            appErr := errors.ToAppError(err)
            
            c.JSON(appErr.StatusCode, gin.H{
                "error": gin.H{
                    "code":    appErr.Code,
                    "message": appErr.Message,
                    "details": appErr.Details,
                },
            })
        }
    }
}
```

- [ ] エラーレスポンスの標準化
- [ ] ログ出力の統一
- [ ] スタックトレースの管理

#### タスク3: バリデーションの改善
- [ ] 入力検証の一元化
- [ ] バリデーションエラーメッセージの改善
- [ ] カスタムバリデータの実装

### Phase 4: テスタビリティの向上（1週間）

#### タスク1: モックとスタブの準備
```go
// backend/internal/mock/session_repository_mock.go
type MockSessionRepository struct {
    mock.Mock
}

func (m *MockSessionRepository) FindByID(ctx context.Context, id string) (*domain.Session, error) {
    args := m.Called(ctx, id)
    return args.Get(0).(*domain.Session), args.Error(1)
}
```

- [ ] gomockまたはtestifyの導入
- [ ] リポジトリモックの生成
- [ ] サービスモックの生成

#### タスク2: テストヘルパーの作成
- [ ] テストデータビルダー
- [ ] テストフィクスチャ
- [ ] テストユーティリティ

#### タスク3: 統合テストの改善
- [ ] テスト用DIコンテナ
- [ ] インメモリデータベースの使用
- [ ] E2Eテストの追加

## 4. マイグレーション戦略

### Step 1: 新旧アーキテクチャの共存
- 既存コードを段階的に移行
- Feature flagでの切り替え
- 後方互換性の維持

### Step 2: 段階的な移行
1. 新規機能は新アーキテクチャで実装
2. 重要度の低い機能から順次移行
3. コアビジネスロジックの移行
4. 完全移行

### Step 3: 旧コードの削除
- デプリケーション警告
- 移行期限の設定
- 最終的な削除

## 5. 実装スケジュール

### Week 1
- ドメイン層の確立
- リポジトリインターフェースの定義

### Week 2
- アプリケーション層の実装
- 依存性注入の導入

### Week 3
- エラーハンドリングの統一
- テスタビリティの向上

### Week 4
- 統合テスト
- ドキュメント作成
- レビューと調整

## 6. 成功基準

- [ ] レイヤー間の依存が単一方向
- [ ] ビジネスロジックの90%以上がドメイン層に集約
- [ ] 単体テストカバレッジ80%以上
- [ ] エラーレスポンスの100%標準化
- [ ] グローバル状態の完全除去

## 7. パフォーマンス指標

### 目標値
- API応答時間: P99 < 200ms
- メモリ使用量: < 500MB
- CPU使用率: < 30%（通常時）

### 計測項目
- レイヤー間の呼び出しオーバーヘッド
- DIコンテナの初期化時間
- エラーハンドリングのオーバーヘッド

## 8. リスクと対策

### リスク1: 大規模リファクタリングによる不具合
- **対策**: 段階的移行と十分なテスト
- **対策**: カナリアリリース

### リスク2: 開発速度の一時的低下
- **対策**: チーム教育とドキュメント整備
- **対策**: ペアプログラミング

### リスク3: オーバーエンジニアリング
- **対策**: YAGNI原則の遵守
- **対策**: 定期的なアーキテクチャレビュー

## 9. ドキュメント計画

- [ ] アーキテクチャ決定記録（ADR）
- [ ] 開発者ガイド
- [ ] APIドキュメント（OpenAPI）
- [ ] デプロイメントガイド

## 10. 長期的な改善項目

### マイクロサービス化の検討
- ログ同期サービスの分離
- ジョブ実行サービスの分離
- 分析サービスの分離

### イベント駆動アーキテクチャ
- ドメインイベントの活用
- CQRS パターンの適用
- イベントソーシングの検討

### 監視とオブザーバビリティ
- 分散トレーシング（OpenTelemetry）
- メトリクス収集（Prometheus）
- 構造化ログ（JSON形式）