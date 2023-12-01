# Acme Shop Users Service

User management and authentication service for the AcmeShop platform.

## Features

- User registration and management
- JWT-based authentication
- Session management with Redis
- Password hashing (bcrypt recommended, MD5/SHA1 legacy support)
- Role-based access control
- Redis caching for user lookups

## API Endpoints

### V2 API (Recommended)

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v2/auth/login` | Authenticate user |
| POST | `/api/v2/auth/logout` | Logout current session |
| POST | `/api/v2/auth/logout/all` | Logout all sessions |
| POST | `/api/v2/auth/refresh` | Refresh JWT token |
| GET | `/api/v2/auth/sessions` | List active sessions |
| DELETE | `/api/v2/auth/sessions/:id` | Revoke session |
| GET | `/api/v2/users` | List users |
| POST | `/api/v2/users` | Create user |
| GET | `/api/v2/users/me` | Get current user |
| PUT | `/api/v2/users/me` | Update current user |
| POST | `/api/v2/users/me/password` | Change password |
| GET | `/api/v2/users/:id` | Get user by ID |
| PUT | `/api/v2/users/:id` | Update user |
| DELETE | `/api/v2/users/:id` | Delete user |

### V1 API (Deprecated)

> **Warning**: V1 API is deprecated and will be removed in v3.0. Please migrate to V2 API.

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/auth/login` | Authenticate user |
| GET | `/api/v1/users` | List users |
| GET | `/api/v1/users/:id` | Get user by ID |
| POST | `/api/v1/users` | Create user |

## Development

### Prerequisites

- Go 1.21+
- PostgreSQL 14+
- Redis 7+

### Setup

```bash
# Clone the repository
git clone https://github.com/tm-acme-shop/acme-shop-users-service.git
cd acme-shop-users-service

# Install dependencies
go mod download

# Set up environment variables
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=acme
export DB_PASSWORD=<password>
export DB_NAME=acme_users
export REDIS_HOST=localhost
export REDIS_PORT=6379
export JWT_SECRET=<secret>

# Run the service
go run ./cmd/users
```

### Testing

```bash
# Run all tests
go test -v ./...

# Run tests with coverage
go test -v -race -coverprofile=coverage.out ./...

# View coverage report
go tool cover -html=coverage.out
```

### Building

```bash
# Build binary
go build -o bin/users-service ./cmd/users

# Build Docker image
docker build -t users-service .
```

## Configuration

Configuration is loaded from environment variables. See `configs/config.yaml` for available options.

### Feature Flags

| Flag | Description | Default |
|------|-------------|---------|
| `ENABLE_LEGACY_AUTH` | Enable legacy MD5 authentication | `false` |
| `ENABLE_V1_API` | Enable deprecated V1 API | `true` |
| `ENABLE_V2_API` | Enable V2 API | `true` |
| `ENABLE_PASSWORD_MIGRATION` | Auto-migrate password hashes | `true` |
| `ENABLE_USER_CACHE` | Enable Redis caching | `true` |
| `ENABLE_DEBUG_MODE` | Enable debug endpoints | `false` |

## Security Notes

### Password Hashing

- **Recommended**: bcrypt (cost factor 12)
- **Deprecated**: MD5, SHA1 (migrated on login when `ENABLE_PASSWORD_MIGRATION=true`)

<!-- TODO(TEAM-SEC): Remove legacy hash support after migration complete -->

### Authentication

- JWT tokens with configurable expiration
- Session tracking in Redis
- Support for session revocation

## Architecture

```
cmd/
  users/           # Application entry point
internal/
  auth/            # Authentication (password, JWT, session)
  config/          # Configuration loading
  handlers/        # HTTP handlers
  migrations/      # Database migrations
  repository/      # Data access layer
  server/          # HTTP server setup
  service/         # Business logic
```

## TODO

- [ ] TODO(TEAM-SEC): Remove MD5/SHA1 password support
- [ ] TODO(TEAM-API): Remove V1 API endpoints
- [ ] TODO(TEAM-PLATFORM): Add OpenTelemetry tracing
- [ ] TODO(TEAM-PLATFORM): Add rate limiting to auth endpoints

## License

Copyright © 2024 AcmeShop. All rights reserved.
