# ACME Shop Users Service

User management service for the ACME Shop platform.

## Features

- User registration and management
- Password authentication (bcrypt for new users, MD5/SHA1 legacy support)
- REST API for user operations (v1 and v2)
- Password hash migration support

## Getting Started

### Prerequisites

- Go 1.21+
- PostgreSQL 14+

### Running locally

```bash
go run ./cmd/users
```

### API Endpoints

#### V1 API (Legacy)
- `GET /api/v1/users` - List all users
- `GET /api/v1/users/:id` - Get user by ID
- `POST /api/v1/users` - Create new user (MD5 hashing)

#### V2 API (New)
- `GET /api/v2/users` - List all users
- `GET /api/v2/users/:id` - Get user by ID
- `POST /api/v2/users` - Create new user (bcrypt hashing)

### Configuration

| Environment Variable | Default | Description |
|---------------------|---------|-------------|
| `ENABLE_LEGACY_AUTH` | `true` | Enable legacy MD5/SHA1 authentication |
| `ENABLE_NEW_AUTH` | `true` | Enable bcrypt authentication |
