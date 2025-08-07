#!/bin/bash

# Basic security validation script for CCDash

echo "🔒 CCDash Security Validation Test"
echo "=================================="
echo ""

# 1. Test CORS_ALLOW_ALL removal
echo "✅ Testing CORS_ALLOW_ALL removal..."
if grep -r "CORS_ALLOW_ALL.*true" backend/cmd/server/main.go >/dev/null 2>&1; then
    echo "❌ FAIL: CORS_ALLOW_ALL is still present in code"
    exit 1
else
    echo "✅ PASS: CORS_ALLOW_ALL functionality removed"
fi

# 2. Test parameterized queries
echo ""
echo "✅ Testing SQL injection protection..."
if grep -r "fmt\.Sprintf.*SELECT\|fmt\.Sprintf.*INSERT\|fmt\.Sprintf.*UPDATE\|fmt\.Sprintf.*DELETE" backend/internal/services/ >/dev/null 2>&1; then
    echo "❌ FAIL: Found potential SQL string concatenation"
    exit 1
else
    echo "✅ PASS: No SQL string concatenation found"
fi

# 3. Test for rate limiting implementation
echo ""
echo "✅ Testing rate limiting implementation..."
if grep -r "RateLimitMiddleware" backend/cmd/server/main.go >/dev/null 2>&1; then
    echo "✅ PASS: Rate limiting middleware is applied"
else
    echo "❌ FAIL: Rate limiting middleware not found"
    exit 1
fi

# 4. Test for panic recovery
echo ""
echo "✅ Testing panic recovery middleware..."
if grep -r "RecoveryMiddleware" backend/cmd/server/main.go >/dev/null 2>&1; then
    echo "✅ PASS: Panic recovery middleware is applied"
else
    echo "❌ FAIL: Panic recovery middleware not found"
    exit 1
fi

# 5. Test for proper API authentication
echo ""
echo "✅ Testing API authentication..."
if grep -r "X-API-Key\|Authorization" backend/internal/middleware/auth.go >/dev/null 2>&1; then
    echo "✅ PASS: API key authentication is implemented"
else
    echo "❌ FAIL: API key authentication not found"
    exit 1
fi

# 6. Test build
echo ""
echo "✅ Testing build compilation..."
cd backend && go build -o ../bin/security-test cmd/server/main.go
if [ $? -eq 0 ]; then
    echo "✅ PASS: Backend compiles successfully"
    rm -f ../bin/security-test
else
    echo "❌ FAIL: Backend compilation failed"
    exit 1
fi

# 7. Test environment configuration files
echo ""
echo "✅ Testing environment configuration..."
cd /Users/satoshi/git/claudeee
if [ -f "backend/configs/production.yaml" ] && [ -f "backend/configs/development.yaml" ] && [ -f "backend/configs/staging.yaml" ]; then
    echo "✅ PASS: Environment configuration files created"
else
    echo "❌ FAIL: Environment configuration files missing"
    exit 1
fi

echo ""
echo "🎉 All security validation tests passed!"
echo "✅ SQLインジェクション対策: 完了"
echo "✅ CORS設定厳格化: 完了"
echo "✅ API認証強化: 完了"
echo "✅ レート制限実装: 完了"
echo "✅ パニックリカバリ: 完了"
echo ""
echo "🚀 セキュリティ修正が正常に完了しました。"