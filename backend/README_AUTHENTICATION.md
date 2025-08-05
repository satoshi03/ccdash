# CCDash Authentication System - Phase 1

This document describes the JWT-based authentication system implemented for CCDash, following the security plan outlined in `plans/authentication-security-plan.md`.

## Overview

Phase 1 implements a comprehensive JWT authentication system with role-based access control (RBAC), audit logging, and rate limiting to secure the CCDash API endpoints.

## Features Implemented

### 🔐 JWT Authentication
- User registration and login
- JWT access tokens (15-minute expiry)
- Refresh tokens (7-day expiry)
- Secure password hashing with bcrypt
- Account lockout after 5 failed login attempts

### 👥 Role-Based Access Control (RBAC)
- **viewer**: Dashboard view only
- **user**: Dashboard + log sync
- **admin**: All permissions including task execution and system management

### 📊 Audit Logging
- All authentication events logged
- Security events tracking
- Failed login attempt monitoring
- Admin activity auditing

### 🚦 Rate Limiting
- API endpoints: 100 requests/minute
- Auth endpoints: 10 requests/minute
- Task execution: 5 requests/minute
- IP-based and user-based limiting

## Configuration

### Environment Variables

```bash
# Enable authentication (default: false for backward compatibility)
AUTH_ENABLED=true

# JWT secret (auto-generated if not provided)
JWT_SECRET=your-secret-key-here

# Optional: CORS settings
CORS_ALLOWED_ORIGINS=https://yourdomain.com
```

### Development Mode

When `AUTH_ENABLED=false` (default), the system runs without authentication for backward compatibility.

### Production Mode

Set `AUTH_ENABLED=true` to enable full authentication and authorization.

## API Endpoints

### Authentication Endpoints

```
POST /api/auth/register         - Register new user
POST /api/auth/login           - Login user
POST /api/auth/refresh         - Refresh access token
POST /api/auth/logout          - Logout user (revoke tokens)
GET  /api/auth/profile         - Get user profile
GET  /api/auth/validate        - Validate current token
```

### Admin Endpoints (admin role required)

```
GET  /api/auth/admin/users/:id        - Get user details
PUT  /api/auth/admin/users/:id/status - Update user status
GET  /api/auth/admin/audit-logs       - Get audit logs
GET  /api/auth/admin/audit-logs/stats - Get audit statistics
```

## Permission Matrix

| Endpoint Group | viewer | user | admin |
|----------------|--------|------|-------|
| Dashboard APIs | ✅ | ✅ | ✅ |
| Log Sync | ❌ | ✅ | ✅ |
| Project Read | ✅ | ✅ | ✅ |
| Project Manage | ❌ | ❌ | ✅ |
| Task Execution | ❌ | ❌ | ✅ |
| User Management | ❌ | ❌ | ✅ |
| Audit Logs | ❌ | ❌ | ✅ |

## Database Schema

### Users Table
```sql
CREATE TABLE users (
    id TEXT PRIMARY KEY,
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    roles TEXT NOT NULL DEFAULT '["user"]', -- JSON array
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_login TIMESTAMP,
    is_active BOOLEAN DEFAULT TRUE,
    failed_login_attempts INTEGER DEFAULT 0,
    locked_until TIMESTAMP NULL
);
```

### Refresh Tokens Table
```sql
CREATE TABLE refresh_tokens (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    token_hash TEXT NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    revoked_at TIMESTAMP NULL,
    is_revoked BOOLEAN DEFAULT FALSE
);
```

### Audit Logs Table
```sql  
CREATE TABLE audit_logs (
    id TEXT PRIMARY KEY,
    user_id TEXT,
    user_email TEXT,
    action TEXT NOT NULL,
    resource TEXT NOT NULL,
    details JSON,
    ip_address TEXT,
    user_agent TEXT,
    success BOOLEAN DEFAULT TRUE,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

## Usage Examples

### Register a User
```bash
curl -X POST http://localhost:6060/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "securepassword123",
    "roles": ["user"]
  }'
```

### Login
```bash
curl -X POST http://localhost:6060/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com", 
    "password": "securepassword123"
  }'
```

### Access Protected Endpoint
```bash
curl -X GET http://localhost:6060/api/token-usage \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN"
```

### Refresh Token
```bash
curl -X POST http://localhost:6060/api/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{"refresh_token": "YOUR_REFRESH_TOKEN"}'
```

## Security Features

### Password Security
- Minimum 8 characters required
- Bcrypt hashing with default cost
- No password stored in plain text

### Account Protection
- Account lockout after 5 failed attempts
- 1-hour lockout duration
- Rate limiting on auth endpoints

### Token Security
- Short-lived access tokens (15 minutes)
- Secure refresh token rotation
- Refresh tokens hashed in database
- Automatic token revocation on logout

### Audit Trail
- All authentication events logged
- Failed login attempts tracked
- Admin actions monitored
- IP address and user agent tracking

## Testing

The authentication system includes comprehensive tests:

```bash
# Run authentication tests
go test ./internal/services -run TestAuthService -v
go test ./internal/middleware -run TestAuthMiddleware -v

# Run specific test categories
go test ./internal/services -run TestAuthService_RegisterUser -v
go test ./internal/services -run TestAuthService_LoginUser -v
go test ./internal/middleware -run TestAuthMiddleware_RequireAuth -v
```

## Migration

### Existing Installations

1. Update the server to latest version
2. Set `AUTH_ENABLED=false` to maintain current behavior
3. When ready to enable auth, set `AUTH_ENABLED=true`
4. Create admin user via registration API
5. Configure frontend to handle authentication

### New Installations

Authentication is disabled by default for easy setup. Enable when ready for production use.

## Next Steps (Phase 2)

- OAuth2/OIDC integration
- Multi-factor authentication (MFA)
- Advanced audit reporting
- Session management improvements
- Container/VM sandboxing for task execution

## Troubleshooting

### Common Issues

1. **"Invalid or expired token"**
   - Check token expiry
   - Verify JWT secret consistency
   - Try refreshing the token

2. **"Account is locked"**
   - Wait 1 hour or reset failed attempts
   - Check audit logs for details

3. **"Insufficient permissions"**
   - Verify user roles
   - Check permission matrix above

### Debug Mode

Enable debug logging:
```bash
export GIN_MODE=debug
```

### Database Issues

Reset authentication tables:
```bash
cd backend/cmd/database-reset && go run main.go
```

## Security Considerations

### Production Deployment

1. **HTTPS Required**: Never deploy without TLS in production
2. **JWT Secret**: Use strong, randomly generated secret
3. **Rate Limiting**: Monitor and adjust limits based on usage
4. **Audit Monitoring**: Set up alerts for suspicious activity
5. **Regular Updates**: Keep dependencies updated

### Network Security

- Deploy behind reverse proxy (nginx)
- Configure proper CORS headers
- Use Web Application Firewall (WAF)
- Implement DDoS protection

### Monitoring

- Monitor failed login attempts
- Track unusual API usage patterns
- Set up alerts for account lockouts
- Regular audit log reviews