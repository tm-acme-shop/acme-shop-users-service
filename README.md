# ACME Shop Users Service

User management service for the ACME Shop platform.

## Features

- User registration and management
- Password authentication (MD5 hashing)
- REST API for user operations

## Getting Started

### Prerequisites

- Go 1.21+
- PostgreSQL 14+

### Running locally

```bash
go run ./cmd/users
```

### API Endpoints

- `GET /health` - Health check
- `GET /api/v1/users` - List all users
- `GET /api/v1/users/:id` - Get user by ID
- `POST /api/v1/users` - Create new user
