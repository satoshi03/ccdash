# HTTPS設定ガイド

## 概要

CCDashをプロダクション環境で使用する場合、セキュリティ上の理由からHTTPS通信を強制することを強く推奨します。特にAPI Key認証を使用する場合、平文での通信は認証情報の漏洩リスクがあります。

## nginx を使用したHTTPS設定

### 1. SSL証明書の準備

Let's Encryptを使用した無料SSL証明書の取得:

```bash
# Certbotのインストール
sudo apt update
sudo apt install certbot python3-certbot-nginx

# SSL証明書の取得
sudo certbot --nginx -d ccdash.example.com
```

### 2. nginx設定ファイル

`/etc/nginx/sites-available/ccdash` を作成:

```nginx
# HTTPからHTTPSへのリダイレクト
server {
    listen 80;
    server_name ccdash.example.com;
    
    # HTTPSへリダイレクト
    return 301 https://$server_name$request_uri;
}

# HTTPS設定
server {
    listen 443 ssl http2;
    server_name ccdash.example.com;
    
    # SSL証明書
    ssl_certificate /etc/letsencrypt/live/ccdash.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/ccdash.example.com/privkey.pem;
    
    # SSL設定
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;
    ssl_prefer_server_ciphers on;
    
    # セキュリティヘッダー
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
    add_header X-Frame-Options "DENY" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;
    add_header Referrer-Policy "strict-origin-when-cross-origin" always;
    
    # フロントエンド
    location / {
        proxy_pass http://localhost:3000;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_cache_bypass $http_upgrade;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
    
    # バックエンドAPI
    location /api {
        # API Key検証 (オプション: nginxレベルでの追加チェック)
        # if ($http_x_api_key = "") {
        #     return 401;
        # }
        
        proxy_pass http://localhost:6060;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        
        # CORS設定（バックエンドで処理される場合は不要）
        # add_header Access-Control-Allow-Origin "https://ccdash.example.com" always;
    }
    
    # 大きなリクエストボディを許可（ログ同期用）
    client_max_body_size 100M;
    
    # タイムアウト設定（長時間実行ジョブ用）
    proxy_connect_timeout 600;
    proxy_send_timeout 600;
    proxy_read_timeout 600;
}
```

### 3. nginx設定の有効化

```bash
# サイトを有効化
sudo ln -s /etc/nginx/sites-available/ccdash /etc/nginx/sites-enabled/

# 設定をテスト
sudo nginx -t

# nginxを再起動
sudo systemctl restart nginx
```

## Apache を使用したHTTPS設定

### 1. 必要なモジュールを有効化

```bash
sudo a2enmod ssl proxy proxy_http headers rewrite
```

### 2. Apache設定ファイル

`/etc/apache2/sites-available/ccdash.conf`:

```apache
<VirtualHost *:80>
    ServerName ccdash.example.com
    # HTTPSへリダイレクト
    RewriteEngine On
    RewriteCond %{HTTPS} off
    RewriteRule ^(.*)$ https://%{HTTP_HOST}$1 [R=301,L]
</VirtualHost>

<VirtualHost *:443>
    ServerName ccdash.example.com
    
    # SSL設定
    SSLEngine on
    SSLCertificateFile /etc/letsencrypt/live/ccdash.example.com/fullchain.pem
    SSLCertificateKeyFile /etc/letsencrypt/live/ccdash.example.com/privkey.pem
    
    # セキュリティヘッダー
    Header always set Strict-Transport-Security "max-age=31536000; includeSubDomains"
    Header always set X-Frame-Options "DENY"
    Header always set X-Content-Type-Options "nosniff"
    Header always set X-XSS-Protection "1; mode=block"
    
    # プロキシ設定
    ProxyRequests Off
    ProxyPreserveHost On
    
    # フロントエンド
    ProxyPass / http://localhost:3000/
    ProxyPassReverse / http://localhost:3000/
    
    # バックエンドAPI
    ProxyPass /api http://localhost:6060/api
    ProxyPassReverse /api http://localhost:6060/api
</VirtualHost>
```

## 環境変数の設定

### バックエンド (.env)

```bash
# HTTPS環境での設定例
GIN_MODE=release
CORS_ALLOWED_ORIGINS=https://ccdash.example.com
```

### フロントエンド (.env.local)

```bash
# HTTPS環境での設定例
NEXT_PUBLIC_API_URL=https://ccdash.example.com/api
```

## セキュリティのベストプラクティス

1. **SSL/TLS証明書の自動更新**
   ```bash
   # Certbotの自動更新設定
   sudo certbot renew --dry-run
   ```

2. **強力なSSL設定**
   - TLS 1.2以上のみを許可
   - 弱い暗号化アルゴリズムを無効化
   - HSTS（HTTP Strict Transport Security）を有効化

3. **追加のセキュリティ対策**
   - ファイアウォールでHTTPポート(80)を閉じる（リダイレクト用を除く）
   - API Keyを定期的にローテーション
   - アクセスログの監視

4. **本番環境チェックリスト**
   - [ ] SSL証明書が正しくインストールされている
   - [ ] HTTPからHTTPSへの自動リダイレクトが機能している
   - [ ] すべてのセキュリティヘッダーが設定されている
   - [ ] API Key認証が有効になっている
   - [ ] CORSが適切に設定されている

## トラブルシューティング

### 証明書エラー
```bash
# 証明書の確認
sudo certbot certificates

# 証明書の手動更新
sudo certbot renew
```

### プロキシエラー
```bash
# nginxエラーログの確認
sudo tail -f /var/log/nginx/error.log

# Apacheエラーログの確認
sudo tail -f /var/log/apache2/error.log
```

### CORS問題
- ブラウザの開発者ツールでCORSエラーを確認
- `CORS_ALLOWED_ORIGINS`環境変数が正しく設定されているか確認