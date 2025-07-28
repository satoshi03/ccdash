# Docker環境でのCCDash実行方法

## 前提条件

- Docker
- Docker Compose
- Claude Code のログファイルが `~/.claude/` に存在すること

## 起動方法

### 1. Docker Composeでの起動

```bash
# プロジェクトルートディレクトリで実行
docker-compose up -d

# または、リアルタイムログを確認しながら起動
docker-compose up
```

### 2. アクセス方法

- **フロントエンド**: http://localhost:3000
- **バックエンドAPI**: http://localhost:8080/api
- **ヘルスチェック**: http://localhost:8080/api/v1/health

### 3. 停止方法

```bash
# コンテナを停止
docker-compose down

# コンテナとボリュームを削除
docker-compose down -v
```

## 個別のDockerコマンド

### バックエンドのみ起動

```bash
cd backend
docker build -t ccdash-backend .
docker run -p 8080:8080 \
  -v ~/.claude:/root/.claude:ro \
  -v ccdash-db:/root/.ccdash \
  ccdash-backend
```

### フロントエンドのみ起動

```bash
cd frontend
docker build -t ccdash-frontend .
docker run -p 3000:3000 \
  -e NEXT_PUBLIC_API_URL=http://localhost:8080/api \
  ccdash-frontend
```

## 設定

### 環境変数

#### バックエンド
- `GIN_MODE`: Ginのモード（`release` または `debug`）
- `PORT`: サーバーポート（デフォルト: 8080）

#### フロントエンド
- `NODE_ENV`: Node.jsの環境（`production` または `development`）
- `NEXT_PUBLIC_API_URL`: バックエンドAPIのURL

### ボリュームマウント

- `~/.claude`: Claude Codeのログディレクトリ（読み取り専用）
- `ccdash-db`: データベース永続化ボリューム

## トラブルシューティング

### ログの確認

```bash
# 全サービスのログを確認
docker-compose logs

# 特定のサービスのログを確認
docker-compose logs backend
docker-compose logs frontend

# リアルタイムログの確認
docker-compose logs -f
```

### データベースのリセット

```bash
# データベースボリュームを削除
docker-compose down -v
docker volume rm ccdash_ccdash-db

# 再起動
docker-compose up -d
```

### Claude Codeログの確認

```bash
# Claude Codeのログディレクトリが存在するか確認
ls -la ~/.claude/projects/

# コンテナ内でのマウント状況を確認
docker exec -it ccdash-backend ls -la /root/.claude/projects/
```

## 開発環境での利用

開発時は以下のコマンドでコンテナを再ビルドできます：

```bash
# キャッシュを使わずに再ビルド
docker-compose build --no-cache

# 特定のサービスのみ再ビルド
docker-compose build --no-cache backend
docker-compose build --no-cache frontend
```

## 注意事項

1. **Claude Codeのログアクセス**: `~/.claude/` ディレクトリが読み取り可能である必要があります
2. **データベース永続化**: データベースは名前付きボリュームで永続化されます
3. **ネットワーク**: フロントエンドとバックエンドは同じDockerネットワーク内で通信します
4. **ポート競合**: ローカルで既に8080または3000ポートが使用されている場合は、docker-compose.ymlのポート設定を変更してください