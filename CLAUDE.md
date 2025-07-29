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

## 更新履歴

- **2025-07-11**: 初回実装完了、基本機能の実装
- **2025-07-11**: README.md、CLAUDE.md作成
- **2025-07-12**: プロジェクト名判定ロジック改善
- **2025-07-25**: SessionWindow計算ロジック修正、時系列表示実装
- **2025-07-25**: ログ同期問題修正、循環依存解決
- **2025-07-25**: コマンドライン管理ツール整備（backend/cmd/配下）
- **2025-07-25**: 不要ファイル削除、プロジェクト構成整理
- **2025-07-25**: リリース手順の標準化、バージョン管理ルール追加

---

*このファイルは開発の進捗に応じて継続的に更新されます。*