# Quick Setup Guide

Get your Health Hub API server up and running in 5 minutes.

## Prerequisites

- Go 1.24+ installed
- PostgreSQL 14+ running (optional for local dev)
- Git

## Step 1: Clone and Navigate

```bash
cd health-app-swift-go/server
```

## Step 2: Create Environment File

```bash
cp .env.example .env
```

Edit `.env` with your settings. For **local development**, minimal config:

```env
# Minimal local setup
APP_ENV=local
PORT=8080

# Auth (dev mode, no real auth required)
AUTH_MODE=dev
AUTH_REQUIRED=0
EMAIL_AUTH_ENABLED=1
JWT_SECRET=local-dev-secret-change-in-production
OTP_DEBUG_RETURN_CODE=1

# Email (print codes to console)
EMAIL_SENDER_MODE=local

# Storage (local filesystem)
BLOB_MODE=local

# AI (mock responses)
AI_MODE=mock

# Database (optional - will use in-memory if not set)
# DATABASE_URL=postgresql://localhost:5432/healthhub?sslmode=disable
```

## Step 3: Install Dependencies

```bash
go mod download
```

## Step 4: Setup Database (Optional)

If using PostgreSQL:

```bash
# Create database
createdb healthhub

# Set connection in .env
DATABASE_URL_POOLED=postgresql://localhost:5432/healthhub?sslmode=disable
DATABASE_URL_DIRECT=postgresql://localhost:5432/healthhub?sslmode=disable

# Run migrations
make migrate-up
# or
go run cmd/migrate/main.go up
```

Without PostgreSQL, the server will use **in-memory storage** (data lost on restart).

## Step 5: Start Server

```bash
make run
# or
go run cmd/api/main.go
```

You should see:

```
========== Health Hub API ==========
  env              = local
  port             = 8080
---- database ----
  runtime_url      = set (via DATABASE_URL_POOLED)
  pooled           = set
...
====================================
Сервер запущен на http://localhost:8080
```

## Step 6: Test It Works

```bash
# Health check
curl http://localhost:8080/healthz

# Request OTP code (watch console for code)
curl -X POST http://localhost:8080/v1/auth/email/request \
  -H 'Content-Type: application/json' \
  -d '{"email": "test@example.com"}'

# Should return: {"status": "ok"}
# Check server console for the OTP code
```

## Common Issues

### Port already in use
```bash
# Kill process on port 8080
lsof -ti:8080 | xargs kill
```

### Database connection error
```bash
# Check PostgreSQL is running
psql -l

# Test connection
psql postgresql://localhost:5432/healthhub
```

### Environment variables not loading
Make sure you're running from `server/` directory:
```bash
cd server && go run cmd/api/main.go  # ✅ Correct
go run server/cmd/api/main.go        # ❌ Wrong - won't find .env
```

## Production Setup

For **staging/production**, you need:

1. **Real database** (not in-memory)
2. **Strong JWT secret** (min 32 chars)
3. **SMTP configured** (for email OTP)
4. **S3/Object storage** (optional, for file uploads)
5. **OpenAI API key** (optional, for AI features)

Example production `.env`:

```env
APP_ENV=production
PORT=8080

# Database (REQUIRED in production)
DATABASE_URL_POOLED=postgresql://user:pass@db.example.com:5432/healthhub?sslmode=require
DATABASE_URL_DIRECT=postgresql://user:pass@db.example.com:5432/healthhub?sslmode=require

# Auth (REQUIRED in production)
AUTH_MODE=siwa  # or dev for email-only
AUTH_REQUIRED=1
JWT_SECRET=<generate-strong-random-secret-min-32-chars>
OTP_SECRET=<another-strong-random-secret>

# Email (REQUIRED if using email auth)
EMAIL_SENDER_MODE=smtp
SMTP_HOST=smtp.yandex.ru
SMTP_PORT=587
SMTP_USERNAME=noreply@yourdomain.com
SMTP_PASSWORD=<app-specific-password>
SMTP_FROM=HealthHub <noreply@yourdomain.com>
SMTP_USE_TLS=1

# Storage (RECOMMENDED for production)
BLOB_MODE=s3
S3_ENDPOINT=https://storage.yandexcloud.net
S3_REGION=ru-central1
S3_BUCKET=your-bucket
S3_ACCESS_KEY_ID=<your-key>
S3_SECRET_ACCESS_KEY=<your-secret>
S3_PUBLIC_BASE_URL=https://storage.yandexcloud.net/your-bucket

# AI (OPTIONAL)
AI_MODE=openai
OPENAI_API_KEY=sk-your-key
OPENAI_MODEL=gpt-4-turbo-preview

# CORS (adjust for your frontend)
CORS_ALLOWED_ORIGINS=https://app.yourdomain.com
CORS_ALLOW_CREDENTIALS=1
```

## Next Steps

- **API Documentation**: See [API.md](./API.md)
- **Troubleshooting**: See [ENV_TROUBLESHOOTING.md](./ENV_TROUBLESHOOTING.md)
- **Authentication Flow**: See [AUTHENTICATION.md](./AUTHENTICATION.md)
- **Database Schema**: See [DATABASE.md](./DATABASE.md)

## Useful Commands

```bash
# Run tests
make test

# Format code
make fmt

# Lint code
make lint

# Build binary
make build

# Migration commands
make migrate-up        # Run all pending migrations
make migrate-status    # Show migration status
make migrate-down      # Rollback last migration (use with caution!)

# Run smoke tests (E2E)
make smoke
```

## Development Workflow

1. Make changes to `.go` files
2. Server auto-reloads on restart (`Ctrl+C` and `make run` again)
3. Add migrations for schema changes (`migrations/` folder)
4. Test endpoints with curl or Postman
5. Check logs for errors

## Environment Variable Priority

When the server loads config:

1. **DATABASE_URL_POOLED** (highest priority for runtime)
2. **DATABASE_URL** (fallback)
3. **DATABASE_URL_DIRECT** (lowest priority, used for migrations)

For migrations, it prefers **DATABASE_URL_DIRECT** over pooled connections.

## Security Notes

- **Never commit `.env`** to version control (already in `.gitignore`)
- Use **app-specific passwords** for SMTP (not your main password)
- Generate **strong random secrets** for production:
  ```bash
  openssl rand -base64 32
  ```
- In production, **AUTH_REQUIRED=1** and **JWT_SECRET** must not be default
- Use **sslmode=require** for database in production

## Getting Help

If you encounter issues:

1. Check startup banner for configuration problems
2. Review server logs: `tail -f /tmp/healthhub.log`
3. See [ENV_TROUBLESHOOTING.md](./ENV_TROUBLESHOOTING.md)
4. Check database connection: `psql $DATABASE_URL`
5. Verify `.env` syntax (no inline comments in values)

---

**Ready to code!** 🚀

For detailed configuration options, see `.env.example`.
