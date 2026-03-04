# CLAOJ v2 - Modern Online Judge Platform

A modern online competitive programming platform built with Go and Next.js.

**Live Site:** [https://claoj.edu.vn](https://claoj.edu.vn)

## Tech Stack

### Frontend (claoj-web)
- **Next.js 16** - React framework with App Router
- **TypeScript** - Type safety
- **Tailwind CSS 4** - Utility-first styling
- **React Query (TanStack Query)** - Data fetching and caching
- **Framer Motion** - Animations
- **next-intl** - Internationalization (English/Vietnamese)
- **React Hook Form + Zod** - Form handling and validation
- **Monaco Editor** - Code editing in browser
- **KaTeX** - Math rendering

### Backend (claoj-go)
- **Go 1.24** - High-performance backend
- **Gin** - HTTP web framework
- **GORM** - ORM for MySQL/MariaDB
- **Redis** - Caching, sessions, and pub/sub
- **JWT** - Authentication
- **WebSockets** - Real-time updates

## Project Structure

```
repo-v2/
├── claoj-web/          # Next.js frontend
│   ├── src/
│   │   ├── app/        # App router pages (i18n with [locale])
│   │   ├── components/ # React components
│   │   ├── lib/        # Utilities and API clients
│   │   ├── types/      # TypeScript types
│   │   ├── i18n/       # Translation files (en.json, vi.json)
│   │   └── utils/      # Helper functions
│   ├── public/         # Static assets
│   ├── __tests__/      # Jest unit tests
│   └── package.json
│
└── claoj-go/           # Go backend
    ├── api/            # HTTP API handlers
    │   └── v2/         # API v2 endpoints
    ├── auth/           # JWT and password utilities
    ├── bridge/         # Judge bridge protocol
    ├── cache/          # Redis caching layer
    ├── config/         # Configuration loading
    ├── db/             # Database connection
    ├── email/          # Email sending
    ├── events/         # WebSocket event system
    ├── jobs/           # Background job processing
    ├── models/         # GORM models
    ├── moss/           # MOSS plagiarism detection
    ├── oauth/          # OAuth2 handlers
    ├── ratelimit/      # Rate limiting middleware
    ├── scoring/        # Rating and scoring algorithms
    └── main.go
```

## Getting Started

### Prerequisites
- Node.js 20+
- Go 1.24+
- MySQL 8.0+ or MariaDB 10.6+
- Redis 7+

### Quick Start

Use the quickstart script:
```bash
./quickstart.sh
```

### Manual Setup

#### Backend

1. Navigate to the backend directory:
```bash
cd claoj-go
```

2. Copy the root `.env.example` to `.env` and configure:
```bash
cp ../../.env.example .env
# Edit .env with your database credentials
```

3. Install dependencies:
```bash
go mod download
```

4. Run the server:
```bash
go run main.go
```

The API will be available at `http://localhost:8080`

#### Frontend

1. Navigate to the frontend directory:
```bash
cd claoj-web
```

2. Copy the example environment file:
```bash
cp .env.example .env
```

3. Install dependencies:
```bash
npm install
```

4. Run the development server:
```bash
npm run dev
```

The frontend will be available at `http://localhost:3000`

## Configuration

### Unified Configuration with `.env`

CLAOJ v2 uses a unified `.env` file for configuration that both backend and frontend can share.
This eliminates duplication and ensures consistency across services.

#### Shared Configuration (.env)

1. Copy the example file at the repository root:
```bash
cp .env.example .env
```

2. Edit `.env` with your values:
```env
# Site Configuration
SITE_URL=http://localhost:3000
API_URL=http://localhost:8080/api/v2
DEFAULT_LANG=en

# Database
DATABASE_URL=user:password@tcp(127.0.0.1:3306)/claoj?charset=utf8mb4&parseTime=True&loc=UTC

# Redis
REDIS_ADDR=127.0.0.1:6379
REDIS_PASSWORD=
REDIS_DB=0

# Server
SERVER_PORT=8080
SERVER_MODE=debug

# Secrets (IMPORTANT: Change in production!)
SECRET_KEY=<generate with: openssl rand -base64 64>

# OAuth (Optional)
OAUTH_GOOGLE_ENABLED=false
OAUTH_GOOGLE_CLIENT_ID=
OAUTH_GOOGLE_CLIENT_SECRET=
```

3. For Docker deployment, place `.env` in the `repo-v2/` directory

#### Environment Variable Priority

The backend loads configuration in this order (highest to lowest priority):

1. Direct environment variables (`DATABASE_URL`, `SECRET_KEY`, etc.)
2. Prefixed environment variables (`CLAOJ_DATABASE_DSN`, `CLAOJ_APP_SECRET_KEY`)
3. `.env` file (loaded automatically via godotenv)
4. Default values

#### Frontend Configuration

The frontend uses `window.location.origin` for API and WebSocket URLs by default.
No environment variables are required for local development.

Optional overrides in `claoj-web/.env` or via Docker Compose environment:
```env
# Only if you need to override defaults
NEXT_PUBLIC_API_URL=http://localhost:8080/api/v2
NEXT_PUBLIC_SITE_NAME=CLAOJ
```

For Docker deployment, set `NEXT_PUBLIC_*` variables in the `.env` file or pass them as build args.

## Features

- Problem solving with multiple programming languages
- Contests with various formats (ICPC, IOI, AtCoder, ECOO)
- Rating system with Elo-based calculations
- Blog posts with voting system
- Real-time notifications via WebSocket
- Bilingual support (English/Vietnamese)
- Modern dark theme UI
- Responsive design
- JWT authentication with OAuth2 support
- MOSS plagiarism detection integration
- Admin dashboard for problem/contest/user management
- Ticket system for support

## Testing

### Backend Tests
```bash
cd claoj-go
go test ./...
```

### Frontend Tests
```bash
cd claoj-web
npm test
```

## Docker Deployment

This repository is designed to work with Docker Compose. See the parent repository
for the full Docker setup including judge servers.

## Migration from v1

The v2 version is a complete rewrite:
- Go backend (replacing Django)
- Next.js frontend (replacing Django templates)
- Improved performance and scalability
- Modern UI/UX with Tailwind CSS
- Better API design with RESTful endpoints

## License

Based on DMOJ - see [LICENSE](LICENSE) file.

## Credits

- Developed by IT-CLA Productions
- Powered by [DMOJ](https://github.com/DMOJ/site) and VNOI
- Long An HSGS Online Judge
