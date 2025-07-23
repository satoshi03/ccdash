# GitHub Actions Setup Guide

このガイドでは、Claudeeeプロジェクトのリリース自動化のためのGitHub Actions設定方法を説明します。

## 必要なSecrets設定

GitHubリポジトリの Settings > Secrets and variables > Actions で以下のSecretsを設定してください：

### 1. NPM_TOKEN
npmへの公開に必要なトークンです。

**取得方法:**
1. [npmjs.com](https://www.npmjs.com)にログイン
2. アカウント設定 > Access Tokens に移動
3. "Generate New Token" > "Automation" を選択
4. トークンをコピーしてGitHub Secretsに `NPM_TOKEN` として設定

### 2. GITHUB_TOKEN
GitHubリリースの作成に使用します（通常は自動的に提供されます）。

## リリースの実行方法

### 1. タグベースのリリース (推奨)
```bash
# タグを作成してプッシュ
git tag v1.0.0
git push origin v1.0.0
```

### 2. 手動リリース
1. GitHubの Actions タブに移動
2. "Build and Release" ワークフローを選択
3. "Run workflow" をクリック
4. バージョン番号を入力（例: 1.0.0）
5. "Run workflow" を実行

## ワークフローの概要

### 1. Backend Build Job
- 5つのプラットフォーム向けにGoバイナリをビルド:
  - macOS Intel (darwin-amd64)
  - macOS Apple Silicon (darwin-arm64)
  - Linux x64 (linux-amd64)
  - Linux ARM64 (linux-arm64)
  - Windows x64 (windows-amd64)

### 2. Frontend Build Job
- Next.jsアプリケーションのビルド
- 静的ファイルの生成

### 3. Create Release Job
- 全バイナリとフロントエンドを統合
- npmパッケージの作成と公開
- GitHubリリースの作成

### 4. Test Package Job
- 複数のプラットフォームでパッケージをテスト
- インストールと動作確認

## トラブルシューティング

### よくある問題

1. **npm公開失敗**
   - NPM_TOKENが正しく設定されているか確認
   - パッケージ名が既に使用されていないか確認

2. **バイナリビルド失敗**
   - Goのバージョンが正しいか確認
   - 依存関係に問題がないか確認

3. **フロントエンドビルド失敗**
   - Node.jsのバージョンが正しいか確認
   - package-lock.jsonが最新か確認

### デバッグ方法

1. Actions タブでワークフローの実行ログを確認
2. 各ステップの詳細なログを確認
3. Artifactsをダウンロードして内容を確認

## パッケージの確認

リリース後、以下の方法でパッケージを確認できます：

```bash
# npm経由でインストール
npm install -g claudeee@latest

# バージョン確認
claudeee version

# ヘルプ表示
claudeee help

# 動作テスト
claudeee --backend-port 8081 --frontend-port 3001
```

## 設定ファイルの場所

- ワークフロー: `.github/workflows/release.yml`
- プラットフォーム検出: `scripts/platform-detector.js`
- インストール後処理: `scripts/postinstall.js`
- npm設定: `package.json`

## 注意事項

1. **タグ名**: `v` プレフィックスを付けてください（例: `v1.0.0`）
2. **バージョン番号**: [Semantic Versioning](https://semver.org/)に従ってください
3. **テスト**: リリース前に手動でビルドとテストを行うことを推奨
4. **バックアップ**: 重要なリリースの前にバックアップを取ってください

## 関連リンク

- [GitHub Actions Documentation](https://docs.github.com/en/actions)
- [npm Publishing Guide](https://docs.npmjs.com/cli/v8/commands/npm-publish)
- [Semantic Versioning](https://semver.org/)