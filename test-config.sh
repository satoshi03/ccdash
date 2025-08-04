#!/bin/bash
# テスト用の環境変数設定

# 別ポートで実行
export PORT=7070

# 別DBパスを使用
export CCDASH_DATABASE_PATH="/tmp/ccdash-test/test.db"

# テスト用ディレクトリ作成
mkdir -p /tmp/ccdash-test

echo "Test configuration:"
echo "  PORT: $PORT"
echo "  DATABASE_PATH: $CCDASH_DATABASE_PATH"
echo ""
echo "Starting backend server..."
cd backend/cmd/server && go run main.go