package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/marcboeker/go-duckdb"
	"github.com/google/uuid"
)

const WINDOW_DURATION = 5 * time.Hour

func roundToNextHour(t time.Time) time.Time {
	return t.Truncate(time.Hour)
}

func main() {
	// データベース接続
	db, err := sql.Open("duckdb", "/Users/satoshi/.claudeee/claudeee.db")
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	fmt.Println("=== Session Windows Migration ===")

	// 1. テーブル作成
	fmt.Println("Creating session_windows table...")
	createTableSQL := `
		CREATE TABLE IF NOT EXISTS session_windows (
			id TEXT PRIMARY KEY,
			window_start TIMESTAMP NOT NULL,
			window_end TIMESTAMP NOT NULL,
			reset_time TIMESTAMP NOT NULL,
			total_input_tokens INTEGER DEFAULT 0,
			total_output_tokens INTEGER DEFAULT 0,
			total_tokens INTEGER DEFAULT 0,
			message_count INTEGER DEFAULT 0,
			session_count INTEGER DEFAULT 0,
			is_active BOOLEAN DEFAULT true,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);

		CREATE INDEX IF NOT EXISTS idx_session_windows_times ON session_windows(window_start, window_end);
		CREATE INDEX IF NOT EXISTS idx_session_windows_active ON session_windows(is_active);
		CREATE INDEX IF NOT EXISTS idx_session_windows_reset_time ON session_windows(reset_time);

		ALTER TABLE messages ADD COLUMN IF NOT EXISTS session_window_id TEXT;
		CREATE INDEX IF NOT EXISTS idx_messages_session_window_id ON messages(session_window_id);
	`

	_, err = db.Exec(createTableSQL)
	if err != nil {
		log.Fatal("Failed to create tables:", err)
	}

	// 2. 既存データからセッションウィンドウを生成
	fmt.Println("Analyzing existing messages to create session windows...")

	// 過去30日のメッセージを取得
	past30Days := time.Now().Add(-30 * 24 * time.Hour)
	
	rows, err := db.Query(`
		SELECT timestamp, session_id, input_tokens, output_tokens
		FROM messages 
		WHERE timestamp >= ?
		ORDER BY timestamp ASC
	`, past30Days)
	if err != nil {
		log.Fatal("Failed to query messages:", err)
	}
	defer rows.Close()

	var currentWindow *SessionWindow
	var windows []*SessionWindow
	messageWindowMap := make(map[string]string) // message timestamp -> window_id

	messageCount := 0
	for rows.Next() {
		var timestamp time.Time
		var sessionID string
		var inputTokens, outputTokens int

		err := rows.Scan(&timestamp, &sessionID, &inputTokens, &outputTokens)
		if err != nil {
			log.Fatal("Failed to scan message:", err)
		}

		messageCount++

		// 現在のウィンドウがない、または現在のメッセージがウィンドウ範囲外の場合
		if currentWindow == nil || timestamp.After(currentWindow.WindowEnd) {
			// 前のウィンドウを完了
			if currentWindow != nil {
				currentWindow.IsActive = false
				windows = append(windows, currentWindow)
			}

			// 新しいウィンドウを作成
			windowStart := timestamp
			windowEnd := roundToNextHour(windowStart.Add(WINDOW_DURATION))
			
			currentWindow = &SessionWindow{
				ID:          uuid.New().String(),
				WindowStart: windowStart,
				WindowEnd:   windowEnd,
				ResetTime:   windowEnd,
				IsActive:    true,
				Sessions:    make(map[string]bool),
			}

			fmt.Printf("Created window %d: %s - %s\n", 
				len(windows)+1,
				windowStart.Format("2006-01-02 15:04:05"), 
				windowEnd.Format("2006-01-02 15:04:05"))
		}

		// メッセージをウィンドウに追加
		currentWindow.TotalInputTokens += inputTokens
		currentWindow.TotalOutputTokens += outputTokens
		currentWindow.TotalTokens += inputTokens + outputTokens
		currentWindow.MessageCount++
		currentWindow.Sessions[sessionID] = true

		messageWindowMap[timestamp.Format(time.RFC3339Nano)] = currentWindow.ID
	}

	// 最後のウィンドウを追加
	if currentWindow != nil {
		// 最後のウィンドウは現在時刻より前なら非アクティブ
		if time.Now().After(currentWindow.WindowEnd) {
			currentWindow.IsActive = false
		}
		windows = append(windows, currentWindow)
	}

	fmt.Printf("Processed %d messages into %d windows\n", messageCount, len(windows))

	// 3. ウィンドウをデータベースに保存
	fmt.Println("Saving session windows to database...")

	tx, err := db.Begin()
	if err != nil {
		log.Fatal("Failed to begin transaction:", err)
	}
	defer tx.Rollback()

	// session_windowsテーブルをクリア
	_, err = tx.Exec("DELETE FROM session_windows")
	if err != nil {
		log.Fatal("Failed to clear session_windows:", err)
	}

	insertWindowSQL := `
		INSERT INTO session_windows (
			id, window_start, window_end, reset_time,
			total_input_tokens, total_output_tokens, total_tokens,
			message_count, session_count, is_active
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	for _, window := range windows {
		_, err = tx.Exec(insertWindowSQL,
			window.ID,
			window.WindowStart,
			window.WindowEnd,
			window.ResetTime,
			window.TotalInputTokens,
			window.TotalOutputTokens,
			window.TotalTokens,
			window.MessageCount,
			len(window.Sessions),
			window.IsActive,
		)
		if err != nil {
			log.Fatal("Failed to insert window:", err)
		}
	}

	// 4. メッセージのsession_window_idを更新
	fmt.Println("Updating messages with session_window_id...")

	// バッチでUPDATEを実行
	var updateCount int
	for timeStr, windowID := range messageWindowMap {
		timestamp, err := time.Parse(time.RFC3339Nano, timeStr)
		if err != nil {
			continue
		}

		result, err := tx.Exec(`
			UPDATE messages SET session_window_id = ? WHERE timestamp = ? AND session_window_id IS NULL
		`, windowID, timestamp)
		if err != nil {
			log.Printf("Warning: Failed to update message at %s: %v", timeStr, err)
			continue
		}

		rowsAffected, _ := result.RowsAffected()
		updateCount += int(rowsAffected)
	}

	err = tx.Commit()
	if err != nil {
		log.Fatal("Failed to commit transaction:", err)
	}

	fmt.Printf("Migration completed successfully!\n")
	fmt.Printf("- Created %d session windows\n", len(windows))
	fmt.Printf("- Updated %d messages with window IDs\n", updateCount)

	// 5. 結果を表示
	fmt.Println("\n=== Session Windows Summary ===")
	
	rows, err = db.Query(`
		SELECT 
			id, window_start, window_end, total_tokens, 
			message_count, session_count, is_active
		FROM session_windows 
		ORDER BY window_start DESC 
		LIMIT 5
	`)
	if err != nil {
		log.Fatal("Failed to query windows:", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id string
		var windowStart, windowEnd time.Time
		var totalTokens, messageCount, sessionCount int
		var isActive bool

		err := rows.Scan(&id, &windowStart, &windowEnd, &totalTokens, &messageCount, &sessionCount, &isActive)
		if err != nil {
			log.Fatal("Failed to scan window:", err)
		}

		status := "Inactive"
		if isActive {
			status = "Active"
		}

		fmt.Printf("Window: %s - %s (%s)\n", 
			windowStart.Format("01/02 15:04"), 
			windowEnd.Format("01/02 15:04"),
			status)
		fmt.Printf("  Tokens: %d, Messages: %d, Sessions: %d\n", 
			totalTokens, messageCount, sessionCount)
		fmt.Println()
	}
}

type SessionWindow struct {
	ID                  string
	WindowStart         time.Time
	WindowEnd           time.Time
	ResetTime           time.Time
	TotalInputTokens    int
	TotalOutputTokens   int
	TotalTokens         int
	MessageCount        int
	Sessions            map[string]bool
	IsActive            bool
}