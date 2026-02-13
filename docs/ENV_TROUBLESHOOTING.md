# Environment Variables Troubleshooting Guide

## Problem: `.env` file not being loaded

### Symptoms
- Server starts with default values (e.g., `auth_mode = none`, `runtime_url = not set`)
- Environment variables from `.env` are ignored
- Application behaves as if running with minimal configuration

### Root Cause
The Go application reads environment variables using `os.Getenv()`, but does not automatically load `.env` files by default.

### Solution
We've added the `godotenv` package with auto-loading functionality:

```go
import (
    _ "github.com/joho/godotenv/autoload"
)
```

This import has been added to:
- `server/cmd/api/main.go` - Main API server
- `server/cmd/migrate/main.go` - Database migrations
- `server/cmd/smoke/main.go` - Smoke tests

The `_` (blank identifier) means the package is imported only for its side effects (loading `.env` file).

### Verification
After starting the server, check the startup banner. You should see:

**Before (broken):**
```
---- database ----
  runtime_url      = not set (will use in-memory storage)
---- auth ----
  auth_mode        = none
  auth_required    = false
```

**After (fixed):**
```
---- database ----
  runtime_url      = set (via DATABASE_URL_POOLED)
---- auth ----
  auth_mode        = dev
  auth_required    = true
```

## Problem: Internal server error on `/v1/auth/email/request`

### Symptoms
```bash
curl -X POST http://localhost:8080/v1/auth/email/request \
  -H 'Content-Type: application/json' \
  -d '{"email": "user@example.com"}'

# Response:
{
  "error": {
    "code": "internal_error",
    "message": "Internal server error"
  }
}
```

### Common Causes

#### 1. Database not initialized
**Solution:** Run migrations first
```bash
make migrate-up
# or
cd server && go run cmd/migrate/main.go up
```

#### 2. Invalid `.env` file syntax
**Common issues:**
- Comments inside quoted values
- Special characters not properly escaped
- Multi-line values without proper escaping

**Example of problematic `.env`:**
```env
# ❌ WRONG - comment inside value
REPORTS_MODE="# local | s3 | auto"

# ✅ CORRECT
REPORTS_MODE=local  # Options: local | s3 | auto
```

#### 3. SMTP configuration issues
Check that SMTP settings are correct:
```env
EMAIL_SENDER_MODE=smtp
SMTP_HOST=smtp.yandex.ru
SMTP_PORT=587
SMTP_FROM=HealthHub <no-reply@yourdomain.com>
SMTP_USERNAME=your-email@example.com
SMTP_PASSWORD=your-app-password
SMTP_USE_TLS=1
```

**Note:** For Yandex/Gmail, use app-specific passwords, not your regular password.

#### 4. In-memory storage fallback
If you see:
```
Ошибка подключения к PostgreSQL: ERROR: relation "profiles" does not exist
Fallback на in-memory storage
```

This means:
- Database exists but tables don't (run migrations)
- Connection string is incorrect
- Database is not accessible

### Testing Email OTP Flow

#### 1. Request OTP code
```bash
curl -X POST http://localhost:8080/v1/auth/email/request \
  -H 'Content-Type: application/json' \
  -d '{"email": "your-email@example.com"}'
```

**Expected response:**
```json
{
  "status": "ok"
}
```

#### 2. Check email or server logs
- In `local` mode with `EMAIL_SENDER_MODE=local`, codes are printed to console
- In SMTP mode, check your email inbox

#### 3. Verify OTP code
```bash
curl -X POST http://localhost:8080/v1/auth/email/verify \
  -H 'Content-Type: application/json' \
  -d '{
    "email": "your-email@example.com",
    "code": "123456"
  }'
```

**Expected response:**
```json
{
  "access_token": "eyJhbG...",
  "token_type": "Bearer",
  "expires_in": 2592000,
  "user_id": "email:your-email@example.com"
}
```

## Debugging Tips

### 1. Check startup banner
Always review the startup banner when the server starts. It shows:
- Which environment variables are loaded
- Which mode each service is using
- Whether secrets are set or using defaults

### 2. View detailed logs
```bash
# Run server with output to file
cd server && go run cmd/api/main.go > /tmp/healthhub.log 2>&1

# Watch logs in real-time
tail -f /tmp/healthhub.log
```

### 3. Check which `.env` file is loaded
The `godotenv/autoload` package looks for `.env` in the current working directory.

Make sure you run commands from the `server/` directory:
```bash
cd server && go run cmd/api/main.go  # ✅ Correct
go run server/cmd/api/main.go        # ❌ Wrong - won't find .env
```

### 4. Verify environment variables are loaded
```bash
# Add temporary debug output in your code
import "os"
log.Printf("DEBUG: DATABASE_URL = %s", os.Getenv("DATABASE_URL"))
```

### 5. Test SMTP connection separately
Create a simple test script to verify SMTP works:
```bash
# Use telnet or openssl to test connection
openssl s_client -connect smtp.yandex.ru:587 -starttls smtp
```

## Quick Fixes Checklist

- [ ] Added `godotenv/autoload` import to main.go
- [ ] Ran `go mod tidy` to install dependencies
- [ ] Ran database migrations (`make migrate-up`)
- [ ] `.env` file is in `server/` directory
- [ ] No syntax errors in `.env` (no inline comments in values)
- [ ] SMTP credentials are correct (use app passwords)
- [ ] Running commands from `server/` directory
- [ ] Server restarts after `.env` changes

## Environment Modes

### Development (`ENV=local`)
- Uses in-memory storage if no database
- Can print OTP codes to console
- Less strict validation
- Debug features enabled

### Staging (`ENV=staging`)
- Requires database connection
- Validates JWT_SECRET is not default
- Requires proper S3 config if `BLOB_MODE=s3`
- SMTP required if `EMAIL_SENDER_MODE=smtp`

### Production (`ENV=production`)
- All validations enforced
- No debug features
- Requires all secrets to be properly set
- Will fail fast if configuration is incomplete

## Common Error Messages

| Error | Cause | Solution |
|-------|-------|----------|
| `runtime_url = not set` | `.env` not loaded | Add godotenv import |
| `relation "profiles" does not exist` | Migrations not run | Run `make migrate-up` |
| `internal_error` on auth endpoints | Multiple possible causes | Check server logs for details |
| `smtp server does not support STARTTLS` | Wrong SMTP port | Use 587 for TLS, 465 for SSL |
| `invalid SMTP_FROM` | Malformed email address | Use format: `Name <email@domain.com>` |
| `otp_resend_too_soon` | Rate limiting active | Wait 60 seconds between requests |

## Getting Help

If issues persist:

1. Check server logs: `tail -f /tmp/healthhub.log`
2. Verify `.env` syntax: `cat server/.env`
3. Test database: `psql $DATABASE_URL -c "SELECT 1"`
4. Test SMTP manually using `telnet` or `openssl s_client`
5. Review startup banner for configuration issues

## Related Documentation

- [Configuration Guide](./CONFIGURATION.md)
- [Database Setup](./DATABASE.md)
- [Authentication Flow](./AUTHENTICATION.md)
