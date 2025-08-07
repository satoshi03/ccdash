# データベース接続管理改善計画

## 優先度: 高（1週間以内対応）

## 1. 現状の問題分析

### 主要な問題点
- 複数箇所で独立した`defer db.Close()`の呼び出し
- 接続プール管理の不在
- 各コマンドツールが独自にDB接続を開く
- 接続リークの可能性
- グローバルな接続管理の欠如

### 影響範囲
- `backend/cmd/` 配下の全コマンドツール（15個以上）
- `backend/internal/services/` の全サービス
- `backend/internal/handlers/` のハンドラー

## 2. 実装計画

### Phase 1: 接続プール管理の実装（2日）

#### タスク1: データベースマネージャーの作成
```go
// backend/internal/database/manager.go
type DatabaseManager struct {
    db          *sql.DB
    config      *Config
    mutex       sync.RWMutex
    metrics     *ConnectionMetrics
}
```

- [ ] シングルトンパターンでの接続管理
- [ ] 接続プールの設定
  - [ ] 最大接続数: 25
  - [ ] アイドル接続数: 5
  - [ ] 接続タイムアウト: 30秒
  - [ ] アイドルタイムアウト: 5分
- [ ] 接続ヘルスチェック機能
- [ ] 自動再接続機能

#### タスク2: 接続メトリクスの実装
- [ ] アクティブ接続数の監視
- [ ] 待機中の接続要求数
- [ ] 接続エラー率の追跡
- [ ] クエリ実行時間の計測

### Phase 2: 依存性注入の導入（2日）

#### タスク1: DIコンテナの実装
```go
// backend/internal/container/container.go
type Container struct {
    dbManager     *database.DatabaseManager
    services      map[string]interface{}
    repositories  map[string]interface{}
}
```

- [ ] wireまたはdigの評価と選定
- [ ] サービス層への注入
- [ ] リポジトリ層への注入
- [ ] ハンドラー層への注入

#### タスク2: 初期化フローの改善
- [ ] アプリケーション起動時の一元的なDB接続
- [ ] Graceful shutdownの実装
- [ ] 接続クリーンアップの保証

### Phase 3: コマンドツールの統合（1日）

#### タスク1: 共通初期化ロジックの作成
```go
// backend/internal/cli/base.go
type BaseCommand struct {
    DB      *sql.DB
    Logger  *log.Logger
    Config  *config.Config
}
```

- [ ] 全コマンドツールの基底構造体
- [ ] 共通の初期化処理
- [ ] エラーハンドリングの統一

#### タスク2: 各コマンドツールの移行
- [ ] database-reset
- [ ] database-status
- [ ] recalculate-windows
- [ ] sync-reset
- [ ] migrate
- [ ] その他全コマンド

### Phase 4: トランザクション管理の改善（1日）

#### タスク1: トランザクションヘルパーの実装
```go
// backend/internal/database/transaction.go
func WithTransaction(ctx context.Context, fn func(*sql.Tx) error) error
```

- [ ] 自動ロールバック機能
- [ ] デッドロック検出と再試行
- [ ] トランザクション分離レベルの管理
- [ ] タイムアウト設定

#### タスク2: 楽観的ロックの実装
- [ ] バージョンカラムの追加
- [ ] 更新競合の検出
- [ ] リトライロジック

## 3. マイグレーション戦略

### Step 1: 新規接続管理システムの並行稼働
- Feature flagでの制御
- 段階的な切り替え
- ロールバック計画

### Step 2: 監視とチューニング
- 接続プールサイズの最適化
- クエリパフォーマンスの測定
- エラー率の監視

### Step 3: 旧システムの削除
- 全機能の動作確認
- パフォーマンステスト
- 本番環境への適用

## 4. テスト計画

### 単体テスト
- [ ] 接続プール管理のテスト
- [ ] トランザクション管理のテスト
- [ ] エラーハンドリングのテスト

### 統合テスト
- [ ] 同時接続数の限界テスト
- [ ] 接続リークの検出テスト
- [ ] フェイルオーバーテスト

### 負荷テスト
- [ ] 高負荷時の接続プール動作
- [ ] 長時間稼働テスト
- [ ] メモリリークの確認

## 5. 監視とアラート

### メトリクス
- データベース接続数
- クエリ実行時間（P50, P90, P99）
- エラー率
- 接続待機時間

### アラート条件
- 接続プール枯渇（80%以上使用）
- クエリタイムアウト（5秒以上）
- 接続エラー率（1%以上）

## 6. 実装スケジュール

### Day 1-2
- データベースマネージャーの実装
- 接続プール管理の実装

### Day 3-4
- 依存性注入の導入
- サービス層の移行

### Day 5
- コマンドツールの統合
- トランザクション管理の改善

### Day 6-7
- テストとデバッグ
- ドキュメント作成
- 本番環境への展開準備

## 7. 成功基準

- [ ] 接続リークゼロ
- [ ] 同時接続数の制御（最大25接続）
- [ ] クエリ応答時間の改善（P99 < 100ms）
- [ ] エラー率 < 0.1%
- [ ] 24時間の連続稼働テスト成功

## 8. リスクと対策

### リスク1: 既存機能への影響
- **対策**: Feature flagによる段階的移行
- **対策**: 包括的な回帰テスト

### リスク2: パフォーマンス劣化
- **対策**: ベンチマークテストの実施
- **対策**: 接続プールサイズの動的調整

### リスク3: DuckDB固有の制約
- **対策**: DuckDBドキュメントの詳細調査
- **対策**: コミュニティサポートの活用

## 9. 参考実装例

```go
// 接続プール設定例
db.SetMaxOpenConns(25)
db.SetMaxIdleConns(5)
db.SetConnMaxLifetime(5 * time.Minute)
db.SetConnMaxIdleTime(5 * time.Minute)

// ヘルスチェック例
ticker := time.NewTicker(30 * time.Second)
go func() {
    for range ticker.C {
        if err := db.Ping(); err != nil {
            log.Error("Database health check failed", err)
            // 再接続ロジック
        }
    }
}()
```

## 10. ドキュメント更新

- [ ] アーキテクチャ図の更新
- [ ] データベース接続ガイドの作成
- [ ] トラブルシューティングガイド
- [ ] パフォーマンスチューニングガイド