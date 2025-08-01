-- Job実行管理テーブル作成マイグレーション
-- 作成日: 2025-08-01
-- 機能: 自動タスク実行機能のジョブ管理

CREATE TABLE jobs (
    id TEXT PRIMARY KEY,                    -- UUID v4
    project_id TEXT NOT NULL,               -- projects.id への外部キー
    command TEXT NOT NULL,                  -- 実行するClaude Codeコマンド
    execution_directory TEXT NOT NULL,      -- 実行ディレクトリの絶対パス
    yolo_mode BOOLEAN DEFAULT FALSE,        -- --yolo フラグの有無
    status TEXT NOT NULL DEFAULT 'pending', -- 'pending', 'running', 'completed', 'failed', 'cancelled'
    priority INTEGER DEFAULT 0,             -- 実行優先度（0=通常、数値が大きいほど高優先）
    created_at TEXT NOT NULL,               -- 作成日時 (ISO8601)
    started_at TEXT,                        -- 実行開始日時
    completed_at TEXT,                      -- 完了日時
    output_log TEXT,                        -- 標準出力ログ
    error_log TEXT,                         -- エラー出力ログ
    exit_code INTEGER,                      -- プロセス終了コード
    pid INTEGER,                           -- 実行中のプロセスID
    -- 将来のスケジュール機能用カラム
    scheduled_at TEXT,                      -- スケジュール実行時刻
    schedule_type TEXT,                     -- 'immediate', 'after_reset', 'custom'
    FOREIGN KEY (project_id) REFERENCES projects(id)
);

-- インデックス作成（クエリ最適化）
CREATE INDEX idx_jobs_project_id ON jobs(project_id);
CREATE INDEX idx_jobs_status ON jobs(status);
CREATE INDEX idx_jobs_created_at ON jobs(created_at DESC);
CREATE INDEX idx_jobs_priority_created ON jobs(priority DESC, created_at DESC);

-- ステータス値の制約（CHECK制約）
-- DuckDBでCHECK制約がサポートされている場合
-- CREATE TABLE jobs_status_check AS 
-- SELECT status FROM jobs WHERE status IN ('pending', 'running', 'completed', 'failed', 'cancelled');