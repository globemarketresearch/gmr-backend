# Healthcare Market Research Backend API

A high-performance, scalable Golang backend API for a Custom Market Insights platform that serves market reports, categories, charts metadata, and analytics summaries with very fast response times (<100ms for cached reads).

## Tech Stack

- **Language**: Go 1.22
- **Web Framework**: Fiber v2
- **Database**: PostgreSQL 16
- **ORM**: GORM
- **Cache**: Redis 7
- **Containerization**: Docker & Docker Compose

## Architecture

This project follows Clean Architecture principles with the following structure:

```
.
├── cmd/
│   └── api/              # Application entry point
├── internal/
│   ├── config/           # Configuration management
│   ├── db/               # Database connection & migrations
│   ├── cache/            # Redis cache layer
│   ├── domain/           # Domain models
│   │   ├── category/
│   │   └── report/
│   ├── repository/       # Data access layer
│   ├── service/          # Business logic layer
│   ├── handler/          # HTTP handlers
│   └── middleware/       # Custom middleware
├── pkg/
│   ├── logger/           # Structured logging
│   ├── response/         # Response helpers
│   └── utils/            # Utility functions
└── migrations/           # SQL migration files
```

## Features (Phase 1)

### Entities
- Category
- Sub-Category
- Market Segment
- Report
- Chart Metadata

### API Endpoints

#### Health Check
- `GET /health` - Health check endpoint

#### Reports
- `GET /api/v1/reports` - Get all reports (paginated)
- `GET /api/v1/reports/:slug` - Get report by slug with full details
- `GET /api/v1/search?q=query` - Search reports

#### Categories
- `GET /api/v1/categories` - Get all categories (paginated)
- `GET /api/v1/categories/:slug` - Get category by slug
- `GET /api/v1/categories/:slug/reports` - Get reports by category (paginated)

### Performance Features
- **Slug-based routing** for SEO-friendly URLs
- **Pagination & cursor support** for large datasets
- **Response time logging** for monitoring
- **GZIP compression** for reduced bandwidth
- **Redis caching** with cache-aside pattern
- **Database indexes** on slug, category_id, and published_at
- **Connection pooling** for optimal database performance
- **Request tracing** with unique request IDs
- **Panic recovery** middleware
- **Graceful shutdown** handling

## Getting Started

### Prerequisites
- Go 1.22 or higher
- Docker and Docker Compose
- PostgreSQL 16 (if running locally)
- Redis 7 (if running locally)

### Installation

1. Clone the repository:
```bash
git clone <repository-url>
cd healthcare-market-research-backend
```

2. Copy the environment file:
```bash
cp .env.example .env
```

3. Update the `.env` file with your configuration

### Running with Docker Compose (Recommended)

Start all services (PostgreSQL, Redis, and API):
```bash
docker-compose up -d
```

View logs:
```bash
docker-compose logs -f api
```

Stop all services:
```bash
docker-compose down
```

### Running Locally

1. Start PostgreSQL and Redis:
```bash
docker-compose up -d postgres redis
```

2. Install dependencies:
```bash
go mod download
```

3. Run the application:
```bash
go run cmd/api/main.go
```

The API will be available at `http://localhost:8081`

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `ENVIRONMENT` | Application environment (development/production) | development |
| `PORT` | Server port | 8081 |
| `DB_HOST` | PostgreSQL host | localhost |
| `DB_PORT` | PostgreSQL port | 5432 |
| `DB_USER` | PostgreSQL user | postgres |
| `DB_PASSWORD` | PostgreSQL password | postgres |
| `DB_NAME` | PostgreSQL database name | healthcare_market |
| `DB_SSLMODE` | PostgreSQL SSL mode | disable |
| `REDIS_HOST` | Redis host | localhost |
| `REDIS_PORT` | Redis port | 6379 |
| `REDIS_PASSWORD` | Redis password | (empty) |
| `REDIS_DB` | Redis database number | 0 |

## API Response Format

### Success Response
```json
{
  "success": true,
  "data": {},
  "meta": {
    "page": 1,
    "limit": 20,
    "total": 100,
    "total_pages": 5
  }
}
```

### Error Response
```json
{
  "success": false,
  "error": "Error message"
}
```

## API Documentation

This API is fully documented with OpenAPI/Swagger specifications.

### Accessing Swagger UI

Once the server is running, you can access the interactive API documentation at:

```
http://localhost:8081/swagger/index.html
```

The Swagger UI provides:
- Complete API endpoint documentation
- Request/response schemas for all endpoints
- Interactive testing capabilities (try out API calls directly from the browser)
- Example requests and responses

### Regenerating Swagger Documentation

If you make changes to API handlers or add new endpoints, regenerate the Swagger docs:

1. Install the swag CLI tool (if not already installed):
```bash
go install github.com/swaggo/swag/cmd/swag@latest
```

2. Generate updated documentation:
```bash
swag init -g cmd/api/main.go -o docs
```

The generated files will be placed in the `docs/` directory.

## Database Schema

The database includes the following tables:
- `categories` - Main categories
- `reports` - Market research reports
- `chart_metadata` - Chart information for reports

All tables include proper indexes for optimal query performance.

## Caching Strategy

- **Cache Provider**: Redis
- **Cache Pattern**: Cache-aside
- **TTL Configuration**:
  - Reports list: 10 minutes
  - Report detail: 30 minutes
  - Categories: 10 minutes

## Development

### Building the Application
```bash
go build -o main ./cmd/api
```

### Running Tests
```bash
go test ./...
```

### Building Docker Image
```bash
docker build -t healthcare-api .
```

## Performance Targets

- P99 latency: < 200ms
- Cached reads: < 100ms
- Database connection pooling enabled
- GZIP compression for responses
- Structured JSON logging

## Next Steps (Future Phases)

- **Phase 2**: Performance optimization with advanced caching and database query optimization
- **Phase 3**: Analytics APIs for trending reports and category analytics
- **Phase 4**: Admin panel with JWT authentication and role-based access control

## License

MIT

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
