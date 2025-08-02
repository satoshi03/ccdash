# データベースマイグレーションフロー

## 概要
サーバー起動時にデータベースファイルが作成される場合のマイグレーションフローについて説明します。

## 一般的なフロー

### 1. 初回起動時（データベースファイルが存在しない場合）

```
1. サーバー起動
2. データベースファイルの存在確認
3. ファイルが存在しない → 新規作成
4. マイグレーションテーブルの作成
5. 全マイグレーションを順番に実行
6. サーバー起動完了
```

### 2. 通常起動時（データベースファイルが存在する場合）

```
1. サーバー起動
2. データベースファイルの存在確認
3. ファイルが存在する → 接続
4. マイグレーションテーブルから適用済みバージョンを確認
5. 未適用のマイグレーションを検出
6. 未適用のマイグレーションを順番に実行
7. サーバー起動完了
```

## マイグレーション管理テーブル

```sql
CREATE TABLE IF NOT EXISTS schema_migrations (
    version VARCHAR PRIMARY KEY,
    applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

## 実装例

### 1. マイグレーションマネージャー

```go
type MigrationManager struct {
    db *sql.DB
    migrations []Migration
}

type Migration struct {
    Version string
    Up      string
    Down    string
}

func (m *MigrationManager) Initialize() error {
    // マイグレーションテーブルの作成
    _, err := m.db.Exec(`
        CREATE TABLE IF NOT EXISTS schema_migrations (
            version VARCHAR PRIMARY KEY,
            applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        )
    `)
    return err
}

func (m *MigrationManager) Migrate() error {
    // 適用済みバージョンの取得
    applied := m.getAppliedVersions()
    
    // 未適用のマイグレーションを実行
    for _, migration := range m.migrations {
        if !contains(applied, migration.Version) {
            if err := m.applyMigration(migration); err != nil {
                return err
            }
        }
    }
    return nil
}
```

### 2. サーバー起動時の統合

```go
func InitializeDatabase(cfg *config.Config) (*sql.DB, error) {
    // データベースディレクトリの作成
    if err := cfg.EnsureDatabaseDir(); err != nil {
        return nil, err
    }
    
    // データベース接続
    db, err := sql.Open("duckdb", cfg.DatabasePath)
    if err != nil {
        return nil, err
    }
    
    // マイグレーションの実行
    migrationManager := NewMigrationManager(db)
    if err := migrationManager.Initialize(); err != nil {
        return nil, err
    }
    if err := migrationManager.Migrate(); err != nil {
        return nil, err
    }
    
    return db, nil
}
```

## ベストプラクティス

### 1. バージョニング
- タイムスタンプベース: `20250802000001_add_jobs_table.sql`
- セマンティックバージョニング: `v1.0.0_add_jobs_table.sql`
- 連番: `001_add_jobs_table.sql`

### 2. トランザクション管理
```go
func (m *MigrationManager) applyMigration(migration Migration) error {
    tx, err := m.db.Begin()
    if err != nil {
        return err
    }
    defer tx.Rollback()
    
    // マイグレーションの実行
    if _, err := tx.Exec(migration.Up); err != nil {
        return err
    }
    
    // バージョンの記録
    if _, err := tx.Exec(
        "INSERT INTO schema_migrations (version) VALUES (?)",
        migration.Version,
    ); err != nil {
        return err
    }
    
    return tx.Commit()
}
```

### 3. ロールバック機能
```go
func (m *MigrationManager) Rollback(version string) error {
    // 指定バージョンまでロールバック
    applied := m.getAppliedVersions()
    
    for i := len(applied) - 1; i >= 0; i-- {
        if applied[i] == version {
            break
        }
        migration := m.findMigration(applied[i])
        if err := m.rollbackMigration(migration); err != nil {
            return err
        }
    }
    return nil
}
```

### 4. 検証機能
```go
func (m *MigrationManager) Validate() error {
    // マイグレーションの整合性チェック
    for i, migration := range m.migrations {
        // バージョンの重複チェック
        for j := i + 1; j < len(m.migrations); j++ {
            if migration.Version == m.migrations[j].Version {
                return fmt.Errorf("duplicate version: %s", migration.Version)
            }
        }
        
        // SQLの構文チェック（オプション）
        if err := m.validateSQL(migration.Up); err != nil {
            return fmt.Errorf("invalid up migration %s: %w", migration.Version, err)
        }
        if err := m.validateSQL(migration.Down); err != nil {
            return fmt.Errorf("invalid down migration %s: %w", migration.Version, err)
        }
    }
    return nil
}
```

## エラーハンドリング

### 1. 部分的な失敗の処理
- トランザクションを使用して原子性を保証
- 失敗時は自動ロールバック
- エラーログの詳細記録

### 2. リトライ機構
```go
func (m *MigrationManager) MigrateWithRetry(maxRetries int) error {
    var err error
    for i := 0; i < maxRetries; i++ {
        if err = m.Migrate(); err == nil {
            return nil
        }
        log.Printf("Migration attempt %d failed: %v", i+1, err)
        time.Sleep(time.Second * time.Duration(i+1))
    }
    return err
}
```

## 開発環境での考慮事項

### 1. 開発用コマンド
```bash
# マイグレーション状態の確認
go run cmd/migrate/main.go status

# マイグレーションの実行
go run cmd/migrate/main.go up

# ロールバック
go run cmd/migrate/main.go down

# 特定バージョンへの移行
go run cmd/migrate/main.go to 20250802000001
```

### 2. テスト環境
- テスト用の別データベースファイルを使用
- 各テストケースでクリーンな状態から開始
- マイグレーションの順序依存性をテスト

## まとめ

このフローにより、以下が実現されます：

1. **自動化**: サーバー起動時に自動的にマイグレーションが実行される
2. **安全性**: トランザクションによる原子性の保証
3. **追跡可能性**: 適用済みマイグレーションの履歴管理
4. **柔軟性**: ロールバック機能による復旧可能性
5. **開発効率**: 開発用コマンドによる手動操作のサポート