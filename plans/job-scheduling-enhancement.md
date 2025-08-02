# ジョブスケジューリング機能拡張計画書

## 概要
現在のジョブ実行機能を拡張し、多様なスケジューリングオプションを提供する。

## 現状分析

### 実装済み機能
1. **基本的なジョブ実行システム**
   - JobService: ジョブのCRUD操作
   - JobExecutor: ジョブの実行とキュー管理
   - 即時実行機能（immediate）
   - 基本的なスケジュールタイプの定義（immediate, after_reset, custom）

2. **データベース構造**
   - jobsテーブル: scheduled_at, schedule_typeカラム実装済み
   - session_windowsテーブル: reset_time情報を保持

3. **フロントエンド**
   - TaskExecutionForm: スケジュールタイプ選択UI（機能は未実装）

### 未実装機能
1. **スケジューリング実装**
   - after_reset: SessionWindowリセット後の実行
   - custom: 時刻指定実行
   - N時間後実行

2. **スケジューラーサービス**
   - 定期的なジョブチェック機能
   - スケジュールされたジョブの実行

3. **UI拡張**
   - 時刻選択UI
   - N時間後指定UI
   - スケジュール詳細表示

## 設計方針

### 1. スケジュールタイプの拡張
```go
const (
    ScheduleTypeImmediate  = "immediate"     // 即時実行
    ScheduleTypeAfterReset = "after_reset"   // ウィンドウリセット後
    ScheduleTypeDelayed    = "delayed"       // N時間後実行（新規）
    ScheduleTypeScheduled  = "scheduled"     // 時刻指定（customを廃止）
)
```

### 2. データベーススキーマ変更
```sql
-- jobsテーブルに追加カラム
ALTER TABLE jobs ADD COLUMN schedule_params TEXT; -- JSON形式でパラメータ保存
```

### 3. スケジュールパラメータ構造
```go
type ScheduleParams struct {
    DelayHours    *int       `json:"delay_hours,omitempty"`    // N時間後実行用
    ScheduledTime *time.Time `json:"scheduled_time,omitempty"` // 時刻指定用
}
```

### 4. JobSchedulerサービス
```go
type JobScheduler struct {
    jobService    *JobService
    jobExecutor   *JobExecutor
    windowService *SessionWindowService
    ticker        *time.Ticker
}
```

## 実装フェーズ

### Phase 1: バックエンド基盤実装（優先度: 高）
**期間**: 2-3日

1. **データベース拡張**
   - schedule_paramsカラム追加
   - マイグレーションスクリプト作成

2. **モデル更新**
   - ScheduleParams構造体定義
   - CreateJobRequestに新フィールド追加
   - Job構造体の拡張

3. **JobServiceの拡張**
   - スケジュールパラメータの保存・取得
   - GetScheduledJobs()メソッド追加
   - 検証ロジックの実装

**テスト項目**:
- スケジュールパラメータの保存・取得
- 各スケジュールタイプの検証
- エラーハンドリング

### Phase 2: スケジューラー実装（優先度: 高）
**期間**: 3-4日

1. **JobSchedulerサービス実装**
   - 定期的なジョブチェック（1分間隔）
   - after_reset実行ロジック
   - delayed実行ロジック
   - scheduled実行ロジック

2. **SessionWindowとの統合**
   - リセット時刻の監視
   - ウィンドウ切り替わり検知

3. **JobExecutorとの統合**
   - スケジュールされたジョブのキューイング

**テスト項目**:
- スケジューラーの定期実行
- 各スケジュールタイプの実行タイミング
- SessionWindowリセット検知
- 同時実行制御

### Phase 3: フロントエンドUI実装（優先度: 中）
**期間**: 2-3日

1. **TaskExecutionFormの拡張**
   - N時間後指定UI（スライダー/数値入力）
   - 日時ピッカーコンポーネント
   - スケジュールプレビュー表示

2. **APIクライアント更新**
   - 新しいリクエスト形式対応
   - バリデーション追加

3. **ジョブ履歴表示の改善**
   - スケジュール情報表示
   - 実行予定時刻表示

**テスト項目**:
- UI操作性
- 入力値検証
- スケジュール情報の正確な表示

### Phase 4: 高度な機能実装（優先度: 低）
**期間**: 2-3日

1. **繰り返し実行**
   - cron式サポート
   - 定期実行設定

2. **スケジュール管理画面**
   - 実行予定ジョブ一覧
   - スケジュール変更・キャンセル

3. **通知機能**
   - 実行開始/完了通知
   - エラー通知

## テスト計画

### 単体テスト
1. **JobService**
   - スケジュールパラメータのCRUD
   - バリデーションロジック

2. **JobScheduler**
   - スケジュール判定ロジック
   - 実行タイミング計算

### 統合テスト
1. **エンドツーエンドテスト**
   - 各スケジュールタイプの動作確認
   - SessionWindowリセットとの連携

2. **負荷テスト**
   - 大量のスケジュールジョブ処理
   - 同時実行制御

### 手動テスト
1. **UI操作テスト**
   - 各種入力パターン
   - エラーケース

2. **実行タイミングテスト**
   - 実際の時間経過での動作確認

## リスクと対策

### 技術的リスク
1. **タイムゾーン問題**
   - 対策: UTCベースで統一、表示時のみローカル変換

2. **スケジューラー停止時の処理**
   - 対策: 起動時に過去のスケジュールをチェック

3. **同時実行制御**
   - 対策: 排他制御とトランザクション管理

### 運用リスク
1. **大量のスケジュールジョブ**
   - 対策: 実行キューの上限設定

2. **長時間実行ジョブ**
   - 対策: タイムアウト設定、強制終了機能

## 実装順序の推奨

1. **Phase 1**: バックエンド基盤（必須）
2. **Phase 2**: スケジューラー実装（必須）
3. **Phase 3**: フロントエンドUI（推奨）
4. **Phase 4**: 高度な機能（オプション）

## 総実装期間
- 最小実装（Phase 1-2）: 5-7日
- 推奨実装（Phase 1-3）: 7-10日
- フル実装（Phase 1-4）: 9-13日

## 次のステップ
1. この計画書のレビューと承認
2. Phase 1の詳細設計作成
3. データベースマイグレーション準備
4. 実装開始