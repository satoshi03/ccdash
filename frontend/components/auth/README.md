# CCDash Authentication Components

フロントエンドでAPIキーを安全に管理するための認証コンポーネント群です。

## コンポーネント

### 1. ApiKeyAuth
ユーザーがAPIキーを入力するためのフォームコンポーネント

```tsx
import { ApiKeyAuth } from '@/components/auth'

<ApiKeyAuth onAuthStateChange={(authenticated) => {
  console.log('認証状態:', authenticated)
}} />
```

**特徴:**
- APIキーの入力・表示/非表示切り替え
- 認証時の自動検証（API呼び出しでテスト）
- sessionStorageへの安全な保存
- エラーハンドリング

### 2. AuthGuard
子コンポーネントを認証で保護するHOCコンポーネント

```tsx
import { AuthGuard } from '@/components/auth'

<AuthGuard>
  <ProtectedContent />
</AuthGuard>
```

**特徴:**
- 認証状態の自動チェック
- 未認証時の認証フォーム表示
- 認証状態のリアルタイム監視

### 3. AuthStatus
現在の認証状態とログアウト機能を提供するコンポーネント

```tsx
import { AuthStatus } from '@/components/auth'

<AuthStatus showFullStatus={true} />
```

**プロパティ:**
- `showFullStatus`: ログアウトボタンを表示するかどうか
- `className`: カスタムCSSクラス

## セキュリティ設計

### APIキーの保存方法
1. **優先度1**: 手動設定（`apiClient.setApiKey()`）
2. **優先度2**: sessionStorage（ブラウザセッション中のみ）
3. **優先度3**: 環境変数（開発時のみ、`NEXT_PUBLIC_API_KEY`）

### セキュリティ機能
- APIキーはsessionStorageに保存（ページ閉じると削除）
- 401エラー時の自動クリア
- 開発時のみ環境変数フォールバック
- HTTPS環境でのセキュアな送信

## 使用方法

### 1. 全アプリケーションの保護
`app/layout.tsx`でAuthGuardを使用:

```tsx
import { AuthGuard } from '@/components/auth'

export default function RootLayout({ children }) {
  return (
    <html>
      <body>
        <AuthGuard>
          {children}
        </AuthGuard>
      </body>
    </html>
  )
}
```

### 2. 個別ページの保護
特定のページのみ保護する場合:

```tsx
import { AuthGuard } from '@/components/auth'

export default function ProtectedPage() {
  return (
    <AuthGuard>
      <div>保護されたコンテンツ</div>
    </AuthGuard>
  )
}
```

### 3. ヘッダーに認証状態表示
```tsx
import { AuthStatus } from '@/components/auth'

export function Header() {
  return (
    <header>
      <AuthStatus showFullStatus={false} />
    </header>
  )
}
```

### 4. 独立認証ページ
`/auth`ページでスタンドアロン認証:

```tsx
// app/auth/page.tsx
import { ApiKeyAuth } from '@/components/auth'

export default function AuthPage() {
  return <ApiKeyAuth onAuthStateChange={handleAuth} />
}
```

## API統合

APIクライアント（`lib/api.ts`）は自動的にAPIキーをヘッダーに追加します:

```typescript
// APIキー設定
apiClient.setApiKey('your-api-key')

// APIキー確認
if (apiClient.hasApiKey()) {
  // リクエスト時に自動的に X-API-Key ヘッダーが追加される
  const data = await apiClient.getTokenUsage()
}

// ログアウト
apiClient.clearApiKey()
```

## 開発時の設定

### 環境変数（開発のみ）
```bash
# .env.local
NEXT_PUBLIC_API_KEY=your-development-api-key
```

**注意**: 本番環境では環境変数を使用せず、必ずユーザー入力を使用してください。

### HTTPS環境での使用
```bash
# .env.https
NEXT_PUBLIC_API_URL=https://localhost/api
NEXT_PUBLIC_API_KEY=your-api-key-for-local-https
```

## トラブルシューティング

### よくある問題

1. **401エラーが発生する**
   - APIキーが正しいか確認
   - バックエンドのAPIキー設定を確認
   - ブラウザのsessionStorageをクリア

2. **認証状態が保持されない**
   - sessionStorageが有効か確認
   - ブラウザがプライベートモードでないか確認

3. **開発環境で環境変数が読み込まれない**
   - `NEXT_PUBLIC_`プレフィックスがあるか確認
   - `.env.local`ファイルが正しい場所にあるか確認

### デバッグ
```typescript
// APIキー状態の確認
console.log('Has API Key:', apiClient.hasApiKey())

// sessionStorageの確認
console.log('Stored Key:', sessionStorage.getItem('ccdash_api_key'))
```