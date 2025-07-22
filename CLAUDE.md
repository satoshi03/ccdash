# Claude Code開発記録

このファイルは、Claude Codeを使用したClaudeeeプロジェクトの開発記録と設定情報を記録するものです。

## プロジェクト概要

**プロジェクト名**: Claudeee  
**目的**: Claude Codeの実行状態をモニタリングし、タスクスケジューリングを行うWebアプリケーション  
**開発開始日**: 2025-07-11  
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

### コマンド集

#### バックエンド開発
```bash
# 開発サーバー起動
cd backend && go run cmd/server/main.go

# ビルド
cd backend && go build -o bin/server cmd/server/main.go

# テスト実行
cd backend && go test ./...

# 依存関係更新
cd backend && go mod tidy
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
ls ~/.claudeee/

# データベースリセット
rm -f ~/.claudeee/claudeee.db*
```

## 実装済み機能

### v1.0.0 (2025-07-11)

#### バックエンド
- [x] Go/Ginによる REST API サーバー
- [x] DuckDB データベース統合
- [x] JSONL ログファイル解析
- [x] トークン使用量計算ロジック
- [x] セッション管理機能
- [x] CORS設定

#### フロントエンド  
- [x] Next.js + TypeScript + Tailwind CSS
- [x] shadcn/ui コンポーネント統合
- [x] レスポンシブデザイン
- [x] API通信フック
- [x] エラーハンドリング
- [x] ローディング状態管理

#### API エンドポイント
- [x] `GET /api/v1/health` - ヘルスチェック
- [x] `GET /api/token-usage` - トークン使用量取得
- [x] `GET /api/claude/sessions/recent` - セッション一覧
- [x] `GET /api/claude/available-tokens` - 利用可能トークン数
- [x] `POST /api/sync-logs` - ログ同期

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
cp ~/.claudeee/claudeee.db ~/.claudeee/claudeee.db.backup

# 2. 既存データベース削除
rm ~/.claudeee/claudeee.db*

# 3. アプリケーション再起動（新ロジックでデータ再生成）
make dev
# または手動でログ同期
curl -X POST http://localhost:8080/api/sync-logs
```

### 判定例

- `/Users/satoshi/git/manavi` → `manavi`
- `/Users/satoshi/git/manavi/backend` → `manavi` (親ディレクトリを使用)
- `/Users/satoshi/git/claude-pilot` → `claude-pilot`

## 今後の実装計画

### v1.1.0 (予定)
- [ ] タスクスケジューリング機能の基本実装
- [ ] 手動タスク実行機能
- [ ] タスクキャンセル機能
- [ ] 優先度設定機能

### v1.2.0 (予定)
- [ ] 自動タスク実行（トークンリセット後）
- [ ] 使用統計・分析機能
- [ ] データエクスポート機能
- [ ] 通知機能

### v2.0.0 (予定)
- [ ] 複数プロジェクト管理
- [ ] ユーザー管理機能
- [ ] 高度な分析・レポート機能
- [ ] API v2実装

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

### 現在の制約
- Claude Codeのログファイル形式に依存
- シングルユーザー前提の実装
- リアルタイム通信未対応

### 今後対応予定の課題
- 大量データ処理時のパフォーマンス
- エラー回復機能の強化
- 設定管理の改善

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
1. **5時間ウィンドウ**: Claude Codeの制限リセット間隔に基づく
2. **JSONL解析**: Claude Codeの出力形式に準拠
3. **プロジェクト名変換**: ハイフン区切りからパス形式への変換ロジック

### デバッグ情報
- データベースファイル: `~/.claudeee/claudeee.db`
- ログファイル: `~/.claude/projects/{project-name}/`
- 設定ファイル: `frontend/.env.local`

### よく使うデバッグコマンド
```bash
# データベース内容確認
# DuckDB CLI で確認が可能

# APIテスト
curl -X GET http://localhost:8080/api/token-usage
curl -X POST http://localhost:8080/api/sync-logs

# プロセス確認
ps aux | grep "go run"
lsof -i :8080
lsof -i :3000
```

## 更新履歴

- **2025-07-11**: 初回実装完了、基本機能の実装
- **2025-07-11**: README.md、CLAUDE.md作成

---

*このファイルは開発の進捗に応じて継続的に更新されます。*