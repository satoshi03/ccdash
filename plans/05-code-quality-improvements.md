# コード品質向上計画

## 優先度: 低〜中（3ヶ月以内対応）

## 1. 現状の問題分析

### 主要な品質問題
1. **テストカバレッジ不足**
   - 統合テストが不足
   - エラーケースのテスト不完全
   - E2Eテストが未実装

2. **ログ出力の問題**
   - console.log/warn/errorの残存
   - 構造化ログの未使用
   - ログレベルの不統一

3. **ドキュメント不足**
   - APIドキュメントが不在
   - 複雑なビジネスロジックの説明不足
   - 開発者向けガイドの欠如

## 2. 品質目標

### 定量的目標
- テストカバレッジ: 80%以上
- 静的解析エラー: 0件
- ドキュメントカバレッジ: 90%以上
- 技術的負債: 20%削減

## 3. 実装計画

### Phase 1: テスト強化（2週間）

#### タスク1: テストフレームワークの整備
```go
// backend/internal/testutil/setup.go
type TestSuite struct {
    DB        *sql.DB
    Server    *gin.Engine
    Client    *http.Client
    Fixtures  *Fixtures
}

func SetupTestSuite(t *testing.T) *TestSuite {
    // テスト用DB初期化
    db := setupTestDB(t)
    
    // テストサーバー作成
    server := setupTestServer(db)
    
    // フィクスチャロード
    fixtures := loadFixtures(db)
    
    return &TestSuite{
        DB:       db,
        Server:   server,
        Client:   &http.Client{},
        Fixtures: fixtures,
    }
}
```

- [ ] テストヘルパー関数の作成
- [ ] フィクスチャ管理システム
- [ ] モック/スタブの整備
- [ ] テストデータビルダー

#### タスク2: 単体テストの充実
```go
// backend/internal/services/session_service_test.go
func TestSessionService_GetRecentSessions(t *testing.T) {
    tests := []struct {
        name      string
        setup     func(*TestSuite)
        limit     int
        want      []*Session
        wantErr   bool
    }{
        {
            name: "正常系: セッション取得",
            setup: func(ts *TestSuite) {
                // テストデータ準備
            },
            limit: 10,
            want:  expectedSessions,
        },
        {
            name: "異常系: DB接続エラー",
            setup: func(ts *TestSuite) {
                ts.DB.Close() // 接続を切断
            },
            wantErr: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            ts := SetupTestSuite(t)
            defer ts.Cleanup()
            
            if tt.setup != nil {
                tt.setup(ts)
            }
            
            service := NewSessionService(ts.DB)
            got, err := service.GetRecentSessions(tt.limit)
            
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
                assert.Equal(t, tt.want, got)
            }
        })
    }
}
```

- [ ] 各サービスの単体テスト作成
- [ ] エッジケースのテスト追加
- [ ] エラーハンドリングのテスト
- [ ] 並行処理のテスト

#### タスク3: 統合テストの実装
```go
// backend/test/integration/api_test.go
func TestAPI_SessionFlow(t *testing.T) {
    ts := SetupIntegrationTest(t)
    defer ts.Cleanup()
    
    // 1. ログ同期
    resp := ts.POST("/api/sync-logs", nil)
    assert.Equal(t, 200, resp.StatusCode)
    
    // 2. セッション一覧取得
    resp = ts.GET("/api/sessions")
    assert.Equal(t, 200, resp.StatusCode)
    
    var sessions []SessionDTO
    json.Unmarshal(resp.Body, &sessions)
    assert.NotEmpty(t, sessions)
    
    // 3. セッション詳細取得
    resp = ts.GET("/api/sessions/" + sessions[0].ID)
    assert.Equal(t, 200, resp.StatusCode)
}
```

- [ ] APIフロー全体のテスト
- [ ] データベース統合テスト
- [ ] 外部サービス連携テスト
- [ ] 認証・認可フローテスト

#### タスク4: E2Eテストの追加
```typescript
// frontend/e2e/dashboard.spec.ts
import { test, expect } from '@playwright/test';

test.describe('Dashboard', () => {
    test('should display session list', async ({ page }) => {
        await page.goto('/');
        
        // 初期化待機
        await page.waitForSelector('[data-testid="session-list"]');
        
        // セッション一覧確認
        const sessions = await page.locator('[data-testid="session-item"]').count();
        expect(sessions).toBeGreaterThan(0);
        
        // セッション詳細表示
        await page.click('[data-testid="session-item"]:first-child');
        await expect(page.locator('[data-testid="session-detail"]')).toBeVisible();
    });
    
    test('should execute job', async ({ page }) => {
        await page.goto('/');
        
        // ジョブ実行フォーム
        await page.fill('[data-testid="command-input"]', 'echo test');
        await page.click('[data-testid="execute-button"]');
        
        // 結果確認
        await expect(page.locator('[data-testid="job-result"]')).toContainText('test');
    });
});
```

- [ ] Playwright or Cypressの導入
- [ ] 主要ユーザーフローのテスト
- [ ] クロスブラウザテスト
- [ ] モバイル対応テスト

### Phase 2: ログシステムの改善（1週間）

#### タスク1: 構造化ログの導入
```go
// backend/internal/logger/logger.go
type Logger struct {
    *zap.Logger
}

func NewLogger(config *Config) *Logger {
    zapConfig := zap.NewProductionConfig()
    
    if config.IsDevelopment() {
        zapConfig = zap.NewDevelopmentConfig()
    }
    
    zapConfig.EncoderConfig.TimeKey = "timestamp"
    zapConfig.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
    
    logger, _ := zapConfig.Build()
    
    return &Logger{logger}
}

// 使用例
logger.Info("Session created",
    zap.String("session_id", session.ID),
    zap.String("project_id", project.ID),
    zap.Duration("duration", duration),
)
```

- [ ] zap or zerologの導入
- [ ] ログレベルの定義
- [ ] コンテキスト情報の付与
- [ ] パフォーマンス影響の最小化

#### タスク2: フロントエンドログの整理
```typescript
// frontend/lib/logger.ts
export class Logger {
    private static instance: Logger;
    private isDevelopment: boolean;
    
    private constructor() {
        this.isDevelopment = process.env.NODE_ENV === 'development';
    }
    
    static getInstance(): Logger {
        if (!Logger.instance) {
            Logger.instance = new Logger();
        }
        return Logger.instance;
    }
    
    info(message: string, data?: any) {
        if (this.isDevelopment) {
            console.log(`[INFO] ${message}`, data);
        }
        // 本番環境では外部サービスに送信
        this.sendToService('info', message, data);
    }
    
    error(message: string, error?: Error, data?: any) {
        console.error(`[ERROR] ${message}`, error, data);
        // エラー追跡サービスへ送信
        this.sendToErrorTracking(error, { message, ...data });
    }
}

export const logger = Logger.getInstance();
```

- [ ] console.*の置き換え
- [ ] ログレベル制御
- [ ] エラー追跡サービス連携
- [ ] ユーザーセッション情報の付与

#### タスク3: ログ集約と分析
- [ ] ログ収集エージェントの設定
- [ ] 中央ログストレージの構築
- [ ] ログ分析ダッシュボード作成
- [ ] アラート設定

### Phase 3: ドキュメント整備（1週間）

#### タスク1: APIドキュメントの作成
```yaml
# api-spec.yaml
openapi: 3.0.0
info:
  title: CCDash API
  version: 1.0.0
  description: Claude Code Dashboard API

paths:
  /api/sessions:
    get:
      summary: Get recent sessions
      parameters:
        - name: limit
          in: query
          schema:
            type: integer
            default: 100
      responses:
        200:
          description: Session list
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: '#/components/schemas/Session'
                  
components:
  schemas:
    Session:
      type: object
      properties:
        id:
          type: string
        project_id:
          type: string
        start_time:
          type: string
          format: date-time
```

- [ ] OpenAPI仕様書の作成
- [ ] Swagger UIの設定
- [ ] APIクライアント生成
- [ ] 使用例の追加

#### タスク2: 開発者ドキュメントの作成
```markdown
# Developer Guide

## Architecture Overview
[アーキテクチャ図]

## Getting Started
### Prerequisites
- Go 1.21+
- Node.js 18+
- DuckDB

### Setup
1. Clone repository
2. Install dependencies
3. Configure environment
4. Run migrations
5. Start development server

## Development Workflow
### Adding New Features
1. Create feature branch
2. Write tests first (TDD)
3. Implement feature
4. Update documentation
5. Submit PR

## Testing
### Running Tests
```bash
# Backend tests
make test-backend

# Frontend tests
make test-frontend

# E2E tests
make test-e2e
```
```

- [ ] README.mdの充実
- [ ] CONTRIBUTING.mdの作成
- [ ] アーキテクチャドキュメント
- [ ] デプロイメントガイド

#### タスク3: コード内ドキュメントの改善
```go
// Package services provides business logic implementations.
package services

// SessionService handles session-related operations.
// It manages the lifecycle of user sessions and provides
// methods for querying and manipulating session data.
type SessionService struct {
    db     *sql.DB
    cache  Cache
    logger Logger
}

// GetRecentSessions retrieves the most recent sessions.
//
// Parameters:
//   - limit: Maximum number of sessions to return (1-1000)
//
// Returns:
//   - []*Session: List of sessions ordered by creation time (newest first)
//   - error: Database error or validation error
//
// Example:
//   sessions, err := service.GetRecentSessions(100)
//   if err != nil {
//       log.Error("Failed to get sessions", err)
//   }
func (s *SessionService) GetRecentSessions(limit int) ([]*Session, error) {
    // Implementation...
}
```

- [ ] GoDocコメントの追加
- [ ] JSDocコメントの追加
- [ ] 複雑なロジックの説明追加
- [ ] 使用例の追加

### Phase 4: 静的解析とリンター（3日）

#### タスク1: Go静的解析の強化
```makefile
# Makefile
.PHONY: lint-backend
lint-backend:
	golangci-lint run --config .golangci.yml
	go vet ./...
	staticcheck ./...
	gosec ./...
```

- [ ] golangci-lintの設定
- [ ] セキュリティ解析（gosec）
- [ ] 複雑度チェック
- [ ] デッドコード検出

#### タスク2: TypeScript/JavaScript解析
```json
// .eslintrc.json
{
  "extends": [
    "next/core-web-vitals",
    "plugin:@typescript-eslint/recommended",
    "plugin:react-hooks/recommended"
  ],
  "rules": {
    "no-console": "error",
    "@typescript-eslint/no-unused-vars": "error",
    "@typescript-eslint/explicit-function-return-type": "warn"
  }
}
```

- [ ] ESLint設定の厳格化
- [ ] Prettierの統一設定
- [ ] TypeScript strictモード
- [ ] import順序の自動整理

#### タスク3: CI/CDパイプラインの強化
```yaml
# .github/workflows/ci.yml
name: CI
on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Lint
        run: make lint
        
      - name: Test
        run: make test
        
      - name: Coverage
        run: make coverage
        
      - name: SonarQube Analysis
        uses: sonarsource/sonarqube-scan-action@master
```

- [ ] 自動テスト実行
- [ ] カバレッジレポート生成
- [ ] 品質ゲートの設定
- [ ] 自動マージブロック

## 4. 品質メトリクス

### 測定項目
- テストカバレッジ率
- 静的解析違反数
- 技術的負債（時間）
- 複雑度（Cyclomatic Complexity）
- 重複コード率

### 目標値
- カバレッジ: 80%以上
- 違反数: 0件
- 技術的負債: 5日以下
- 複雑度: 10以下
- 重複率: 3%以下

## 5. 実装スケジュール

### Month 1
- Week 1-2: テスト強化
- Week 3: ログシステム改善
- Week 4: ドキュメント整備

### Month 2
- Week 1: 静的解析導入
- Week 2-3: リファクタリング
- Week 4: CI/CD強化

### Month 3
- Week 1-2: 品質改善の継続
- Week 3: レビューと調整
- Week 4: 最終確認とリリース

## 6. 成功基準

- [ ] テストカバレッジ80%達成
- [ ] 全API仕様書完成
- [ ] 開発者ガイド完成
- [ ] 静的解析エラー0件
- [ ] CI/CDパイプライン完全自動化

## 7. リスクと対策

### リスク1: 開発速度の低下
- **対策**: 段階的な品質向上
- **対策**: 自動化の活用

### リスク2: 過度なテスト
- **対策**: 重要度に基づく優先順位付け
- **対策**: コスト効果の評価

### リスク3: ドキュメント腐敗
- **対策**: 自動生成の活用
- **対策**: レビュープロセスに組み込み