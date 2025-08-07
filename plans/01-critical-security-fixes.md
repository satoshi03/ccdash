# 重大セキュリティ脆弱性修正計画

## 優先度: 最高（1週間以内対応）

## 1. SQLインジェクション対策

### 現状の問題
- 動的SQLクエリ構築時にパラメータ化が不完全
- 直接的な文字列結合によるクエリ生成が存在

### 実装タスク

#### Phase 1: 既存クエリの監査（1日）
- [ ] 全データベースアクセスコードの洗い出し
- [ ] 脆弱なクエリパターンの特定
- [ ] リスク評価とリファクタリング優先順位決定

#### Phase 2: パラメータ化クエリへの移行（2-3日）
- [ ] `backend/internal/services/` 配下の全サービスのクエリ修正
  - [ ] session_service.go
  - [ ] project_service.go
  - [ ] token_service.go
  - [ ] session_window_service.go
  - [ ] diff_sync_service.go
- [ ] `backend/cmd/` 配下のコマンドツールのクエリ修正
- [ ] プリペアドステートメントの使用徹底

#### Phase 3: クエリビルダーの導入検討（2日）
- [ ] sqlx または squirrel の評価
- [ ] 移行計画の策定
- [ ] パイロット実装

### 検証項目
- [ ] SQLインジェクションテストの実施
- [ ] パフォーマンステスト
- [ ] 回帰テスト

## 2. API認証の強化

### 現状の問題
- API認証が環境変数に依存
- 開発モードで認証を無効化可能
- APIキーがフロントエンド経由で露出リスク

### 実装タスク

#### Phase 1: 認証メカニズムの改善（2日）
- [ ] JWTベースの認証実装
  - [ ] トークン生成・検証ロジック
  - [ ] リフレッシュトークン機構
  - [ ] トークン有効期限管理
- [ ] セッション管理の実装
  - [ ] セッションストレージ（Redis or メモリ）
  - [ ] セッションタイムアウト

#### Phase 2: APIキー管理の改善（1日）
- [ ] APIキーの暗号化保存
- [ ] キーローテーション機能
- [ ] レート制限の実装
- [ ] APIキー使用履歴の記録

#### Phase 3: フロントエンド側の対策（1日）
- [ ] セキュアなトークン保存（httpOnly Cookie）
- [ ] CSRF対策の実装
- [ ] XSS対策の強化

### 検証項目
- [ ] ペネトレーションテスト
- [ ] 認証フローのセキュリティ監査
- [ ] トークン漏洩シナリオのテスト

## 3. CORS設定の厳格化

### 現状の問題
- `CORS_ALLOW_ALL=true`で全オリジン許可可能
- プライベートIPアドレスの自動許可

### 実装タスク

#### Phase 1: CORS設定の見直し（半日）
- [ ] 許可オリジンの明示的定義
- [ ] 環境別設定ファイルの作成
  - [ ] development.yaml
  - [ ] staging.yaml
  - [ ] production.yaml
- [ ] ワイルドカード許可の削除

#### Phase 2: 動的CORS管理（半日）
- [ ] 許可オリジンのデータベース管理
- [ ] 管理UIの実装
- [ ] 監査ログの実装

### 検証項目
- [ ] 各環境でのCORS動作確認
- [ ] 不正オリジンからのアクセステスト
- [ ] プリフライトリクエストの確認

## 実装スケジュール

### Day 1-2
- SQLインジェクション対策の監査と修正開始
- CORS設定の厳格化

### Day 3-4
- SQLインジェクション対策の完了
- API認証メカニズムの実装

### Day 5-6
- APIキー管理の改善
- フロントエンド側のセキュリティ対策

### Day 7
- 統合テスト
- セキュリティ監査
- ドキュメント更新

## 成功基準
- [ ] OWASP Top 10の脆弱性チェックをパス
- [ ] 静的解析ツールでのセキュリティ警告ゼロ
- [ ] ペネトレーションテストでの重大な問題なし

## リスクと対策
- **リスク**: 既存機能の破壊
  - **対策**: 段階的な移行とfeature flagの使用
- **リスク**: パフォーマンス劣化
  - **対策**: ベンチマークテストの実施
- **リスク**: 開発速度の低下
  - **対策**: 開発環境用の簡易認証モードの維持

## 参考資料
- [OWASP SQL Injection Prevention Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/SQL_Injection_Prevention_Cheat_Sheet.html)
- [OWASP Authentication Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Authentication_Cheat_Sheet.html)
- [OWASP CORS Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Cross-Origin_Resource_Sharing_Cheat_Sheet.html)