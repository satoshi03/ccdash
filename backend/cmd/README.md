# CCDash Backend Commands

このディレクトリには、CCDashバックエンドの管理とメンテナンス用のコマンドラインツールが含まれています。

## 利用可能なコマンド

### server
メインのAPIサーバーを起動します。
```bash
cd cmd/server && go run main.go
```

### database-reset
データベースを完全にリセットします。すべてのデータが削除されます。
```bash
cd cmd/database-reset && go run main.go
```
- 使用場面: データベースの整合性問題が発生した場合、クリーンな状態から始めたい場合

### sync-reset
ファイル同期状態をリセットして、すべてのJSONLファイルを最初から再処理します。
```bash
cd cmd/sync-reset && go run main.go
```
- 使用場面: 同期プロセスに問題がある場合、新しい解析ロジックを適用したい場合

### database-status
現在のデータベースの状態を表示します。
```bash
cd cmd/database-status && go run main.go
```
- 表示内容: セッション数、メッセージ数、セッションウィンドウ数、トークン数、最近の活動

### recalculate-windows
既存のメッセージを基にセッションウィンドウを再計算します。
```bash
cd cmd/recalculate-windows && go run main.go
```
- 使用場面: セッションウィンドウの計算ロジックを変更した後、既存データに新しいロジックを適用したい場合

### fix-session-times
セッションの開始時刻と終了時刻を修正します。
```bash
cd cmd/fix-session-times && go run main.go
```

### migrate-session-windows
セッションウィンドウテーブルのマイグレーションを実行します。
```bash
cd cmd/migrate-session-windows && go run main.go
```

## 一般的な使用パターン

### 問題のトラブルシューティング
1. `database-status` でデータベースの状態を確認
2. 問題があれば `sync-reset` で同期状態をリセット
3. 重大な問題があれば `database-reset` で完全リセット

### 新しいロジックの適用
1. `sync-reset` でファイル同期をリセット
2. サーバーを再起動してログを再処理
3. `recalculate-windows` でセッションウィンドウを再計算

### 開発・テスト時のクリーンアップ
1. `database-reset` でデータベースをクリア
2. サーバーを起動して自動的にデータベーススキーマを作成
3. ログ同期を実行してテストデータを読み込み

## ヘルプ
各コマンドは `--help` オプションでヘルプを表示できます：
```bash
cd cmd/<command-name> && go run main.go --help
```