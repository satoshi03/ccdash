# CCDash - Claude Code Dashboard

このファイルは、Claude Codeを使用したCCDashプロジェクトの開発記録と設定情報を記録するものです。

## プロジェクト概要

**プロジェクト名**: CCDash  
**目的**: Claude Codeのログを解析し、セッション管理・トークン使用量監視を行うWebアプリケーション  
**開発開始日**: 2025-07-11  
**現在のバージョン**: v1.0.0 (基本機能実装完了)  
**Claude Codeバージョン**: 4.0+  

## 開発環境

### 必要なツール・環境
- Go 1.21以上
- Node.js 18以上
- npm/pnpm
- Git

### 推奨設定

#### VS Code拡張機能
- Go (golang.go)
- TypeScript and JavaScript Language Features
- Tailwind CSS IntelliSense
- ES7+ React/Redux/React-Native snippets

#### Git設定
```bash
git config --global user.name "Your Name"
git config --global user.email "your.email@example.com"
```

## 開発ワークフロー

### 新機能開発時の手順

1. **要件確認**
   - 機能仕様の明確化
   - 技術要件の整理
   - パフォーマンス要件の確認

2. **設計フェーズ**
   - API設計
   - データベーススキーマ設計
   - UI/UXデザイン

3. **実装フェーズ**
   - バックエンドAPI実装
   - フロントエンドコンポーネント実装
   - 統合テスト

4. **テスト・デバッグ**
   - 単体テスト
   - 統合テスト
   - エラーハンドリング確認

### リリース手順

#### バージョンアップとタグ作成
リリースタグを作成する際は、必ずpackage.jsonのバージョンも同時に更新してください：

```bash
# 1. package.jsonのバージョンを更新
# 例: "version": "0.1.5" → "version": "0.1.6"

# 2. 変更をコミット
git add package.json
git commit -m "Bump version to 0.1.6"

# 3. メインブランチにプッシュ
git push origin main

# 4. リリースタグを作成
git tag -a v0.1.6 -m "v0.1.6 - リリース内容の説明"

# 5. タグをプッシュ
git push origin v0.1.6
```

**重要**: package.jsonのバージョンとGitタグのバージョンは必ず一致させること。npm publishで使用されるため、バージョン不整合があるとCI/CDでエラーが発生します。

### コマンド集

#### バックエンド開発
```bash
# 開発サーバー起動
cd backend/cmd/server && go run main.go

# ビルド
cd backend && go build -o bin/server cmd/server/main.go

# テスト実行
cd backend && go test ./...

# 依存関係更新
cd backend && go mod tidy

# データベース状態確認
cd backend/cmd/database-status && go run main.go

# SessionWindow再計算
cd backend/cmd/recalculate-windows && go run main.go

# 同期状態リセット
cd backend/cmd/sync-reset && go run main.go

# データベース完全リセット
cd backend/cmd/database-reset && go run main.go
```

#### フロントエンド開発
```bash
# 開発サーバー起動
cd frontend && npm run dev

# ビルド
cd frontend && npm run build

# 型チェック
cd frontend && npm run type-check

# リント
cd frontend && npm run lint

# 依存関係更新
cd frontend && npm update
```

#### データベース
```bash
# データベースファイルの場所
ls ~/.ccdash/

# データベースリセット
rm -f ~/.ccdash/ccdash.db*
```

## 実装済み機能

### v1.0.0 (2025-07-11 - 2025-07-25)

#### バックエンド
- [x] Go/Ginによる REST API サーバー
- [x] DuckDB データベース統合
- [x] JSONL ログファイル解析（差分同期対応）
- [x] トークン使用量計算ロジック
- [x] 5時間ウィンドウによるセッション管理
- [x] CORS設定
- [x] SessionWindow 計算ロジック（時系列順、重複排除）
- [x] メッセージとSessionWindowの自動関連付け
- [x] ファイル同期状態管理

#### フロントエンド  
- [x] Next.js + TypeScript + Tailwind CSS
- [x] shadcn/ui コンポーネント統合
- [x] レスポンシブデザイン
- [x] API通信フック
- [x] エラーハンドリング
- [x] ローディング状態管理
- [x] SessionWindow時系列表示

#### API エンドポイント
- [x] `GET /api/v1/health` - ヘルスチェック
- [x] `GET /api/token-usage` - トークン使用量取得
- [x] `GET /api/claude/sessions/recent` - セッション一覧
- [x] `GET /api/claude/available-tokens` - 利用可能トークン数
- [x] `POST /api/sync-logs` - ログ同期
- [x] `GET /api/claude/session-windows` - SessionWindow一覧（時系列順）

#### コマンドラインツール
- [x] `backend/cmd/server` - APIサーバー起動
- [x] `backend/cmd/database-status` - データベース状態確認
- [x] `backend/cmd/database-reset` - データベース完全リセット
- [x] `backend/cmd/sync-reset` - ファイル同期状態リセット
- [x] `backend/cmd/recalculate-windows` - SessionWindow再計算
- [x] `backend/cmd/fix-session-times` - セッション時刻修正
- [x] `backend/cmd/migrate-session-windows` - セッションウィンドウマイグレーション

## SessionWindow計算ロジックの改善 (2025-07-25)

### SessionWindow計算アルゴリズム

Claude Codeの5時間制限リセットに基づく適切なSessionWindow計算を実装：

1. **最古のメッセージから開始**: データベース内の最古のメッセージを取得
2. **5時間ウィンドウ作成**: メッセージ時刻から5時間のウィンドウを作成
3. **時刻の丸め処理**:
   - WindowStart: 分単位で切り捨て（例：10:23 → 10:23）
   - WindowEnd: 時間単位で切り捨て（例：15:23 → 15:00）
   - ResetTime: WindowEndと同じ（時間単位切り捨て）
4. **順次ウィンドウ作成**: 次の未割り当てメッセージで新しいウィンドウを作成
5. **重複排除**: 同じ時間範囲のウィンドウは作成しない

### 実装されたSessionWindow機能

- 時系列順でのSessionWindow表示
- メッセージの自動SessionWindow割り当て
- 循環依存問題の解決
- 差分ログ同期でのSessionWindow計算

### トラブルシューティング用コマンド

```bash
# SessionWindow再計算
cd backend/cmd/recalculate-windows && go run main.go

# データベース状態確認
cd backend/cmd/database-status && go run main.go

# 同期状態リセット
cd backend/cmd/sync-reset && go run main.go
```

## プロジェクト名の改善について (2025-07-12)

### 改善されたプロジェクト名判定ロジック

セッションログの `cwd` フィールドを使用してより正確なプロジェクト名を取得するよう改善しました：

- **従来**: ディレクトリ名をハイフン区切りに変換 (`-Users-satoshi-git-manavi`)
- **改善後**: `cwd` から実際のプロジェクト名を抽出 (`manavi`)
- **サブディレクトリ対応**: `frontend`, `backend`, `src`, `lib` などのサブディレクトリの場合は親ディレクトリ名を使用

### 新ロジックを適用する方法

既存データに新しいプロジェクト名判定を適用するには：

```bash
# 1. データベースバックアップ
cp ~/.ccdash/ccdash.db ~/.ccdash/ccdash.db.backup

# 2. 既存データベース削除
rm ~/.ccdash/ccdash.db*

# 3. アプリケーション再起動（新ロジックでデータ再生成）
make dev
# または手動でログ同期
curl -X POST http://localhost:6060/api/sync-logs
```

### 判定例

- `/Users/satoshi/git/manavi` → `manavi`
- `/Users/satoshi/git/manavi/backend` → `manavi` (親ディレクトリを使用)
- `/Users/satoshi/git/claude-pilot` → `claude-pilot`

## 今後の実装計画

### v1.1.0 (検討中)
- [ ] プロジェクト別の詳細分析機能
- [ ] トークン使用量の時系列グラフ
- [ ] セッション継続時間の分析
- [ ] エクスポート機能（CSV/JSON）

### v1.2.0 (検討中)
- [ ] リアルタイムログ監視
- [ ] Webソケット通信による即座更新
- [ ] アラート機能（使用量上限など）
- [ ] 複数プロジェクト管理UI

### v2.0.0 (将来構想)
- [ ] 多用户对应
- [ ] 认证・认可机制
- [ ] 高度分析和报告功能
- [ ] API v2实现
- [ ] 插件系统

## 技術的な決定事項

### データベース選択
- **選択**: DuckDB
- **理由**: 
  - 軽量で高性能
  - SQLite互換
  - 分析クエリに適している
  - 依存関係が少ない

### フロントエンド技術選択
- **選択**: Next.js + TypeScript
- **理由**:
  - 型安全性
  - SSR/SSG対応
  - 開発体験が良い
  - エコシステムが充実

### UIライブラリ選択
- **選択**: shadcn/ui
- **理由**:
  - カスタマイズ性が高い
  - Tailwind CSSとの統合が良い
  - アクセシビリティ対応
  - 最新のReact慣習に従っている

## パフォーマンス考慮事項

### バックエンド
- データベースクエリの最適化
- 大量のJSONLファイル処理の効率化
- メモリ使用量の監視

### フロントエンド
- 仮想化による大きなリストの最適化
- 適切なキャッシュ戦略
- バンドルサイズの最適化

## セキュリティ考慮事項

### データ保護
- ローカルファイルのアクセス制御
- ログファイルの機密情報除去
- CORS設定の適切な管理

### 認証・認可
- 将来的な多ユーザー対応時の認証方式
- API エンドポイントのセキュリティ

## 運用・保守

### ログ管理
- アプリケーションログの出力先
- エラーログの監視
- パフォーマンス監視

### バックアップ
- データベースファイルのバックアップ戦略
- 設定ファイルのバージョン管理

## 既知の制約・課題

## 現在の実装状況

### 動作確認済み機能
- ✅ Claude CodeのJSONLログ解析
- ✅ SessionWindow自動計算・時系列表示
- ✅ トークン使用量集計
- ✅ セッション管理
- ✅ 差分ログ同期
- ✅ Web UI による監視
- ✅ 管理コマンドツール

### 現在の制約
- Claude Codeのログファイル形式に依存
- シングルユーザー前提の実装
- リアルタイム通信未対応（手動同期が必要）

### 今後対応予定の課題
- 大量データ処理時のパフォーマンス最適化
- リアルタイムログ監視機能
- 複数プロジェクト同時管理UI

## 参考資料・リンク

### 技術ドキュメント
- [Go Documentation](https://golang.org/doc/)
- [Gin Framework](https://gin-gonic.com/docs/)
- [DuckDB Documentation](https://duckdb.org/docs/)
- [Next.js Documentation](https://nextjs.org/docs)
- [shadcn/ui Documentation](https://ui.shadcn.com/)

### Claude Code関連
- [Claude Code Documentation](https://docs.anthropic.com/claude/docs)
- [Claude Code GitHub Repository](https://github.com/anthropics/claude-code)

## 開発メモ

### 重要な設計決定
1. **5時間SessionWindow**: Claude Codeの制限リセット間隔に基づく時系列ウィンドウ管理
2. **JSONL差分解析**: Claude Codeの出力形式に準拠し、差分同期で効率的処理
3. **プロジェクト名変換**: cwdフィールドからの正確なプロジェクト名抽出
4. **時刻丸め処理**: WindowStartは分単位、WindowEnd/ResetTimeは時間単位で切り捨て
5. **循環依存解決**: SessionWindow計算時の依存関係問題の解決
6. **DuckDB採用**: 分析処理に適した高性能データベース

### デバッグ情報
- データベースファイル: `~/.ccdash/ccdash.db`
- ログファイル: `~/.claude/projects/{project-name}/`
- 設定ファイル: `frontend/.env.local`

### よく使うデバッグコマンド
```bash
# データベース状態確認
cd backend/cmd/database-status && go run main.go

# SessionWindow状態確認
curl -X GET http://localhost:6060/api/claude/session-windows

# ログ同期実行
curl -X POST http://localhost:6060/api/sync-logs

# トークン使用量確認
curl -X GET http://localhost:6060/api/token-usage

# プロセス確認
ps aux | grep "go run"
lsof -i :6060
lsof -i :3000

# データベースリセット（問題がある場合）
cd backend/cmd/database-reset && go run main.go

# 同期状態リセット
cd backend/cmd/sync-reset && go run main.go
```

## セキュリティ設定の変更 (2025-08-05)

### コマンド安全性チェック機能への移行

従来の固定ホワイトリスト方式から、Claude Codeを使用した動的安全性チェック方式に変更しました。

#### 新しい環境変数

```bash
# コマンド安全性チェック設定（旧ホワイトリスト設定から変更）
CCDASH_DISABLE_SAFETY_CHECK=false  # 安全性チェックを無効化
CCDASH_CLAUDE_CODE_PATH=claude     # Claude Codeのパス（デフォルト: claude）
```

#### 動作の変更点

1. **従来**: 事前定義されたコマンドのみ許可
2. **新方式**: Claude Codeが各コマンドの危険性を動的に分析
3. **利点**: 自然言語コマンドに対応、柔軟性向上
4. **安全性**: 明らかに安全なコマンドは即座に許可、不明なコマンドはAI分析

#### 実行方法

```bash
# コマンド安全性チェック実行例
claude --print "以下のコマンドについて安全性をチェックしてください: npm install express"
```

#### 重要な仕様

- **安全性チェックは各ジョブの実行ディレクトリで行われます**
- ジョブごとに異なるプロジェクトディレクトリのコンテキストで判定
- プロジェクトの構成や依存関係を考慮した安全性判定が可能

#### 安全性チェックの無効化方法

安全性チェックが不安定な場合やテスト時に無効化する複数の方法：

1. **環境変数で無効化（.env推奨）**
   ```bash
   # .envファイルに追加
   CCDASH_DISABLE_SAFETY_CHECK=true
   ```

2. **NPMスクリプトで一時的に無効化**
   ```bash
   # 安全性チェックなしで開発サーバー起動
   npm run dev:no-safety
   
   # API認証なしで開発サーバー起動
   npm run dev:no-auth
   
   # 両方とも無効化（開発専用）
   npm run dev:unsafe
   ```

3. **NPXコマンドオプションで無効化**
   ```bash
   # 安全性チェック無効化
   npx ccdash --no-safety
   
   # API認証無効化
   npx ccdash --no-auth
   
   # 両方とも無効化
   npx ccdash --no-safety --no-auth
   ```

4. **環境変数で一時的に無効化**
   ```bash
   # 一回限りの実行
   CCDASH_DISABLE_SAFETY_CHECK=true npm run dev
   ```

5. **シェルスクリプトで無効化**
   ```bash
   ./backend/scripts/run-no-safety.sh
   ```

⚠️ **注意**: 安全性チェックを無効化すると、すべてのコマンドが検証なしで実行されます。信頼できる環境でのみ使用してください。

## API Key自動生成機能 (2025-08-06)

### セキュアなAPI Key管理

CCDashは初回起動時に自動的に安全なAPI Keyを生成・管理します。

#### 主要機能

1. **自動生成**: 256bit暗号学的に安全なランダムキー
2. **自動保存**: `.env`ファイルへの永続化
3. **セキュア表示**: 本番環境では省略表示
4. **環境別動作**: 開発/本番モードで適切な動作

#### 実装された機能

- `backend/internal/config/api_key_manager.go`: API Key管理コア機能
- `backend/internal/config/api_key_manager_test.go`: 包括的テストスイート
- `backend/internal/middleware/auth.go`: 認証ミドルウェア統合

#### 動作例

```bash
# 初回起動時（開発モード）
🔐 New API key generated!
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
🔑 API Key: a1b2c3d4e5f6...890f1234567890abcdef1234567890ab
⚠️  Development mode: Full key displayed above
💾 Key saved to: .env
🔧 Use this key for API authentication
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

# 本番モード
🔑 API Key: a1b2c3d4...7890 (full key in .env file)
🔒 Production mode: Key truncated for security
```

#### セキュリティ設計

- **暗号学的安全性**: crypto/randを使用した真の乱数生成
- **適切な長さ**: 256bit（64文字hex）で十分な強度
- **安全な表示**: 本番では先頭8文字+末尾4文字のみ表示
- **ファイル保護**: 適切なファイル権限で.envファイル作成

⚠️ **注意**: 安全性チェックを無効化すると、すべてのコマンドが検証なしで実行されます。信頼できる環境でのみ使用してください。

#### 削除されたファイル

- `backend/internal/services/command_whitelist.go`
- `backend/internal/services/command_whitelist_test.go`

#### 新規追加ファイル

- `backend/internal/services/command_safety_checker.go`
- `backend/internal/services/command_safety_checker_test.go`

## 更新履歴

- **2025-07-11**: 初回実装完了、基本機能の実装
- **2025-07-11**: README.md、CLAUDE.md作成
- **2025-07-12**: プロジェクト名判定ロジック改善
- **2025-07-25**: SessionWindow計算ロジック修正、時系列表示実装
- **2025-07-25**: ログ同期問題修正、循環依存解決
- **2025-07-25**: コマンドライン管理ツール整備（backend/cmd/配下）
- **2025-07-25**: 不要ファイル削除、プロジェクト構成整理
- **2025-07-25**: リリース手順の標準化、バージョン管理ルール追加
- **2025-08-05**: コマンドホワイトリストからClaude Code安全性チェック機能に移行
- **2025-08-06**: API Key自動生成・管理機能追加、NPXコマンドオプション拡張

---

*このファイルは開発の進捗に応じて継続的に更新されます。*