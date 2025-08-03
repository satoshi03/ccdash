# CCDash 認証・セキュリティ計画

## 概要
タスク実行機能の実装により、システムへの不正アクセスリスクが大幅に増加します。特に、nginxなどを使用してインターネットに公開する場合、適切な認証・認可メカニズムが必須となります。

## リスク分析

### タスク実行機能に関するリスク
1. **コマンドインジェクション**
   - 任意のコマンド実行が可能になる危険性
   - OSレベルでの破壊的操作の実行リスク

2. **権限昇格**
   - アプリケーション権限を超えた操作の実行
   - システムファイルへの不正アクセス

3. **リソース枯渇攻撃**
   - 無限ループや重い処理による DoS
   - ディスク容量の枯渇

4. **情報漏洩**
   - 環境変数や設定ファイルの露出
   - システム情報の不正取得

### ベーシック認証の限界
1. **平文パスワード送信** (HTTPS必須)
2. **セッション管理なし**
3. **ブルートフォース攻撃への脆弱性**
4. **ログアウト機能の欠如**
5. **細かい権限制御不可**

## 推奨認証方式

### Phase 1: 最小限のセキュリティ (ローカル/信頼できる環境)
```yaml
認証方式: API Key認証
実装:
  - 環境変数でAPIキーを設定
  - HTTPヘッダーでキーを送信
  - HTTPS必須
メリット:
  - 実装が簡単
  - ステートレス
デメリット:
  - キー漏洩リスク
  - ユーザー管理なし
```

### Phase 2: 基本的なユーザー管理 (小規模チーム)
```yaml
認証方式: JWT (JSON Web Token)
実装:
  - ユーザー登録・ログイン機能
  - JWTトークン発行・検証
  - リフレッシュトークン対応
  - ロール基本管理 (admin/user)
メリット:
  - セッション管理
  - 細かい権限制御可能
  - スケーラブル
デメリット:
  - 実装の複雑性
  - トークン管理必要
```

### Phase 3: エンタープライズレベル (公開環境)
```yaml
認証方式: OAuth2.0/OpenID Connect
実装:
  - 外部IDプロバイダー連携 (Google, GitHub等)
  - または Keycloak などの認証サーバー
  - MFA (多要素認証) 対応
  - 監査ログ
メリット:
  - 最高レベルのセキュリティ
  - SSO対応
  - コンプライアンス対応
デメリット:
  - 実装・運用の複雑性
  - 外部依存
```

## 実装計画

### Step 1: API Key認証 (即座に実装可能)
```go
// backend/internal/middleware/auth.go
type AuthMiddleware struct {
    apiKey string
}

func (a *AuthMiddleware) Authenticate() gin.HandlerFunc {
    return func(c *gin.Context) {
        key := c.GetHeader("X-API-Key")
        if key == "" || key != a.apiKey {
            c.JSON(401, gin.H{"error": "unauthorized"})
            c.Abort()
            return
        }
        c.Next()
    }
}
```

### Step 2: JWT認証 (1-2週間)
```go
// backend/internal/auth/jwt.go
type JWTManager struct {
    secretKey     []byte
    tokenDuration time.Duration
}

type Claims struct {
    UserID string   `json:"user_id"`
    Email  string   `json:"email"`
    Roles  []string `json:"roles"`
    jwt.StandardClaims
}

// ユーザーテーブル追加
CREATE TABLE users (
    id TEXT PRIMARY KEY,
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    roles TEXT NOT NULL, -- JSON array
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_login TIMESTAMP
);
```

### Step 3: タスク実行の権限制御
```go
// backend/internal/middleware/rbac.go
type Permission string

const (
    PermissionViewDashboard   Permission = "dashboard:view"
    PermissionSyncLogs       Permission = "logs:sync"
    PermissionExecuteTasks   Permission = "tasks:execute"
    PermissionManageSystem   Permission = "system:manage"
)

type RolePermissions map[string][]Permission

var DefaultRoles = RolePermissions{
    "viewer": {PermissionViewDashboard},
    "user":   {PermissionViewDashboard, PermissionSyncLogs},
    "admin":  {PermissionViewDashboard, PermissionSyncLogs, PermissionExecuteTasks, PermissionManageSystem},
}
```

## セキュリティ強化策

### 1. タスク実行の制限
```yaml
実行制限:
  - ホワイトリスト方式のコマンド許可
  - サンドボックス環境での実行
  - タイムアウト設定
  - リソース制限 (CPU/メモリ)
  
許可コマンド例:
  - git status/diff/log (読み取り専用)
  - npm/yarn list
  - go mod graph
  - 特定ディレクトリ内のみ操作可能
```

### 2. 監査ログ
```sql
CREATE TABLE audit_logs (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    action TEXT NOT NULL,
    resource TEXT NOT NULL,
    details JSON,
    ip_address TEXT,
    user_agent TEXT,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### 3. レート制限
```go
// backend/internal/middleware/ratelimit.go
func RateLimitMiddleware() gin.HandlerFunc {
    // タスク実行: 1分間に5回まで
    // API呼び出し: 1分間に100回まで
}
```

### 4. HTTPS設定 (nginx)
```nginx
server {
    listen 443 ssl http2;
    server_name ccdash.example.com;
    
    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;
    
    # セキュリティヘッダー
    add_header Strict-Transport-Security "max-age=31536000" always;
    add_header X-Frame-Options "DENY" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;
    
    location / {
        proxy_pass http://localhost:3000;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }
    
    location /api {
        proxy_pass http://localhost:6060;
        # APIキー検証
        if ($http_x_api_key = "") {
            return 401;
        }
    }
}
```

## 実装優先順位

### 即座に実装すべき (Phase 0)
1. **API Key認証** - 最低限の保護
2. **HTTPS強制** - 通信の暗号化
3. **実行コマンドのホワイトリスト** - 危険なコマンドの防止

### 短期的に実装 (Phase 1: 1-2週間)
1. **JWT認証システム**
2. **基本的なRBAC (Role-Based Access Control)**
3. **監査ログ**
4. **レート制限**

### 中長期的に検討 (Phase 2: 1-2ヶ月)
1. **OAuth2.0/OIDC統合**
2. **MFA (多要素認証)**
3. **セッション管理の高度化**
4. **コンテナ/VM でのサンドボックス実行**

## 環境別推奨設定

### ローカル開発環境
- API Key認証で十分
- HTTPSは必須ではない
- 実行コマンド制限は緩め

### 社内/チーム環境
- JWT認証推奨
- HTTPS必須
- VPN経由のアクセスに限定
- 厳格なコマンドホワイトリスト

### 公開環境
- OAuth2.0/OIDC必須
- WAF (Web Application Firewall) 導入
- DDoS対策
- 定期的なセキュリティ監査
- ペネトレーションテスト

## まとめ

タスク実行機能の実装においては、最低限 **API Key認証 + HTTPS + コマンドホワイトリスト** の組み合わせが必須です。公開環境では、より高度な認証システムと多層防御が必要となります。

段階的に実装を進めることで、開発速度を維持しながらセキュリティを強化できます。