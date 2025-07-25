-- セッションウィンドウ管理テーブル
CREATE TABLE IF NOT EXISTS session_windows (
    id TEXT PRIMARY KEY,
    window_start TIMESTAMP NOT NULL,           -- ウィンドウ開始時刻（最初のメッセージ時刻）
    window_end TIMESTAMP NOT NULL,             -- ウィンドウ終了時刻（開始+5時間、時刻切り捨て）
    reset_time TIMESTAMP NOT NULL,            -- 実際のリセット時刻（window_end）
    total_input_tokens INTEGER DEFAULT 0,      -- このウィンドウ内の合計入力トークン
    total_output_tokens INTEGER DEFAULT 0,     -- このウィンドウ内の合計出力トークン
    total_tokens INTEGER DEFAULT 0,           -- このウィンドウ内の合計トークン
    message_count INTEGER DEFAULT 0,          -- このウィンドウ内のメッセージ数
    session_count INTEGER DEFAULT 0,          -- このウィンドウ内のセッション数
    is_active BOOLEAN DEFAULT true,           -- アクティブウィンドウかどうか
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- インデックス作成
CREATE INDEX IF NOT EXISTS idx_session_windows_times ON session_windows(window_start, window_end);
CREATE INDEX IF NOT EXISTS idx_session_windows_active ON session_windows(is_active);
CREATE INDEX IF NOT EXISTS idx_session_windows_reset_time ON session_windows(reset_time);

-- メッセージテーブルにsession_window_idカラムを追加
ALTER TABLE messages ADD COLUMN IF NOT EXISTS session_window_id TEXT;
CREATE INDEX IF NOT EXISTS idx_messages_session_window_id ON messages(session_window_id);

-- 外部キー制約（DuckDBでサポートされている場合）
-- ALTER TABLE messages ADD CONSTRAINT fk_messages_session_window 
-- FOREIGN KEY (session_window_id) REFERENCES session_windows(id);