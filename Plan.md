# 🚀 High-Performance Golang API for Custom Market Insights

---

## 🎯 Objective

Build a **high-performance, scalable Golang backend API** for a Custom Market Insights platform that serves market reports, categories, charts metadata, and analytics summaries with very fast response times (**<100ms** for cached reads).

The API will power a **Next.js frontend** and must be optimized for:

- ⚡ Heavy read traffic
- 🔍 SEO-friendly report pages
- 📊 Large but mostly static datasets
- 🤖 Future AI / analytics extensions

---

## 🧠 Core Principles

- ⚡ **Performance first**
- 🏗️ **Clean Architecture**
- 📈 **Horizontal scalability**
- 💾 **Cache-heavy reads**
- 🚀 **Minimal cold starts**
- 🔄 **Stateless APIs**

---

## 🛠️ Tech Stack

| Component | Technology |
|-----------|-----------|
| **Language** | Go (latest stable) |
| **Web Framework** | Fiber (preferred) or net/http (with chi) |
| **Database** | PostgreSQL |
| **ORM** | GORM (use raw SQL for hot paths) |
| **Cache** | Redis |
| **Serialization** | JSON (optimize with struct tags) |
| **Auth** | JWT (Phase 3) |
| **Deployment** | Docker |
| **Images** | Cloudinary (API only, no storage logic) |

---

## 📁 Project Structure

Following **Clean Architecture** principles:

```
/cmd
  /api                    # Application entry point
/internal
  /config                 # Configuration management
  /db                     # Database connection
  /cache                  # Redis cache layer
  /domain                 # Business entities
    /report
    /category
    /analytics
  /repository             # Data access layer
  /service                # Business logic layer
  /handler                # HTTP handlers
  /middleware             # HTTP middleware
/pkg
  /logger                 # Structured logging
  /response               # Standard API responses
  /utils                  # Utility functions
/migrations               # Database migrations
```

---

## 📦 Phase 1: Core API (Read-Optimized)

### 📋 Entities

- **Report**
- **Category**
- **Sub-Category**
- **Market Segment**
- **Chart Metadata** (no chart images)

### 🔌 API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/reports` | List all reports (paginated) |
| `GET` | `/reports/:slug` | Get report by slug |
| `GET` | `/categories` | List all categories |
| `GET` | `/categories/:slug/reports` | Get reports by category |
| `GET` | `/search?q=` | Search reports |

### ✅ Requirements

- ✓ Slug-based routing
- ✓ Pagination & cursor support
- ✓ Response time logging
- ✓ GZIP compression
- ✓ Database indexes on:
  - `slug`
  - `category_id`
  - `published_at`

---

## ⚡ Phase 2: Performance Optimization

### 💾 Caching Strategy

**Redis Cache:**
- Report list
- Report detail by slug
- Category → report mapping

**Cache TTL:**
| Resource | TTL |
|----------|-----|
| Reports list | 10 minutes |
| Report detail | 30 minutes |

**Patterns:**
- ✓ Use **cache-aside pattern**
- ✓ Use **singleflight** to prevent cache stampede

### 🗄️ Database Optimization

- ✓ Use connection pooling
- ✓ Use prepared statements
- ✓ Use raw SQL for read-heavy endpoints
- ✓ Avoid N+1 queries
- ✓ Implement eager loading where appropriate

---

## 📊 Phase 3: Analytics APIs

### 🔌 API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/analytics/report/:id` | Report analytics |
| `GET` | `/analytics/category/:id` | Category analytics |
| `GET` | `/analytics/trending` | Trending reports |

### 📝 Notes

- Aggregated queries
- Cached aggressively
- No real-time writes needed

---

## 🔐 Phase 4: Admin & Auth (Optional)

### 🛡️ Authentication

- JWT authentication
- Role-based access control (admin/editor/viewer)

### 🔌 Admin API Endpoints

- `POST /admin/reports` - Create report
- `PUT /admin/reports/:id` - Update report
- `PATCH /admin/reports/:id/publish` - Publish report
- `PATCH /admin/reports/:id/unpublish` - Unpublish report

### ⚠️ Cache Invalidation

- Writes **invalidate Redis cache** automatically

---

## 🔎 Non-Functional Requirements

| Requirement | Target |
|-------------|--------|
| **P99 Latency** | < 200ms |
| **Graceful Shutdown** | ✓ Required |
| **Structured Logging** | ✓ Required |
| **Request Tracing** | ✓ Middleware |
| **Panic Recovery** | ✓ Middleware |
| **Health Check** | `GET /health` |

---

## 🧪 Testing

- ✅ Unit tests for services
- ✅ Integration tests for repositories
- ✅ Load-test ready (k6 compatible)

---

## 🐳 Deployment

- ✅ Dockerfile (multi-stage build)
- ✅ ENV-based configuration
- ✅ Ready for:
  - Railway
  - Fly.io
  - Kubernetes

---

## 🚧 Important Constraints

| Constraint | Details |
|------------|---------|
| ❌ No SSR | API only (frontend handled separately) |
| ❌ No image uploads | Use Cloudinary URLs only |
| ⚖️ Read/Write Ratio | Optimize for **90:10** (read > write) |

---

## ✅ Deliverables

1. ✅ Fully runnable Golang API
2. ✅ Clear README with setup instructions
3. ✅ Sample `.env` file
4. ✅ Example Redis + PostgreSQL setup
5. ✅ Benchmark results for key endpoints

---

## 🎁 Bonus Features (If Time Permits)

- 🏷️ ETag support
- 📦 HTTP caching headers (Cache-Control, Expires)
- 📖 OpenAPI/Swagger documentation
- 🛡️ Rate limiting middleware

---

**Last Updated:** December 2024
**Status:** 📋 Planning Phase