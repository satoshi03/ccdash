# CCDash

A monitoring dashboard for Claude Code usage and session management.

![CCDash Screenshot](assets/images/ccdash_top_1.png)

## Quick Start

```bash
# Run instantly
npx ccdash

# Or install globally
npm install -g ccdash
ccdash
```

Access at: http://localhost:3000

## Features

- **Token Usage Monitoring** - Track usage within 5-hour windows with reset times
- **Session Management** - View sessions by project with token usage and execution time
- **Automatic Sync** - Parse Claude Code logs and sync to database

## Development Setup

### Prerequisites
- Go 1.21+
- Node.js 18+

### Installation
```bash
git clone <repository-url>
cd ccdash

# Backend
cd backend && go mod download

# Frontend  
cd frontend && npm install --legacy-peer-deps
```

## Commands

```bash
# Production
npx ccdash

# Development  
npx ccdash dev

# Custom frontend port
npx ccdash --frontend-port 3001

# Custom frontend URL
npx ccdash --frontend-url https://app.example.com

# Other commands
npx ccdash build
npx ccdash help
npx ccdash version
```

**Note**: Backend runs on port 6060 (fixed)

## Tech Stack

**Backend**: Go, Gin, DuckDB  
**Frontend**: Next.js, TypeScript, Tailwind CSS, shadcn/ui

## Manual Setup (Development)

```bash
# Backend
cd backend && go run cmd/server/main.go

# Frontend (new terminal)
cd frontend && npm run dev
```

## Production Deployment

For production with custom domains, use nginx reverse proxy (see `nginx/README.md`)

## Troubleshooting

### Common Issues

**Database Lock Error**
```bash
rm -f ~/.ccdash/ccdash.db*
```

**Frontend Dependency Error**  
```bash
cd frontend && npm install --legacy-peer-deps
```

**CORS Error**  
Use nginx reverse proxy (see `nginx/README.md`) or use localhost with custom ports.

## Security Configuration (Phase 0)

CCDash now includes basic security features to protect your deployment:

### API Key Authentication

Protect your API endpoints with API key authentication:

```bash
# Backend
export CCDASH_API_KEY=your-secret-key-here

# Frontend
export NEXT_PUBLIC_API_KEY=your-secret-key-here
```

In development mode (default), authentication is disabled. For production:
- Set `GIN_MODE=release` to enforce authentication
- Generate a strong API key (e.g., `openssl rand -hex 32`)

### Command Whitelist

The job execution feature includes a command whitelist to prevent dangerous operations:

**Allowed by default:**
- Git read-only commands (`git status`, `git diff`, `git log`)
- Package managers read-only (`npm list`, `yarn list`, `go list`)
- Testing commands (`npm test`, `go test`, `pytest`)
- Linting and formatting (`eslint`, `prettier`, `go fmt`)
- Development servers (`npm run dev`, `yarn dev`)

**Configuration:**
```bash
# Add custom commands
export CCDASH_ALLOWED_COMMANDS=custom-tool,another-tool

# Disable whitelist (NOT recommended)
export CCDASH_DISABLE_COMMAND_WHITELIST=true
```

### HTTPS Setup

For production deployments, use HTTPS to encrypt all communications:

See [nginx/README.md](nginx/README.md) for complete HTTPS setup guide with nginx configuration.

### Environment Variables

Copy the example files and configure:
```bash
cp backend/.env.example backend/.env
cp frontend/.env.example frontend/.env.local
```

See the `.env.example` files for all available options.

## License

MIT License
