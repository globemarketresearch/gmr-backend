# Reports API Documentation

Complete API request/response specification for all report-related endpoints.

## Base Configuration

- **Base URL**: Configured in `lib/config.ts` (typically `/api`)
- **Authentication**: JWT Bearer token (auto-refreshed)
- **Content-Type**: `application/json`
- **Response Format**: All responses wrapped in `ApiResponse<T>`

---

## Standard Response Wrapper

All API responses follow this structure:

```typescript
{
  "success": boolean,
  "data"?: T,
  "error"?: string,
  "meta"?: {
    "page"?: number,
    "limit"?: number,
    "total"?: number,
    "total_pages"?: number,
    "cursor"?: string
  }
}
```

---

## 1. Get All Reports

### Endpoint
```
GET /api/v1/reports
```

### Authentication
❌ Not required (public endpoint)

### Query Parameters
```typescript
{
  // Public query parameters
  page?: number;          // Page number (default: 1)
  limit?: number;         // Items per page (default: 10, max: 50)
  status?: "draft" | "published";
  category?: string;      // Category name
  geography?: string;     // Geography filter
  accessType?: "free" | "paid";
  search?: string;        // Search query

  // Admin-only query parameters (requires admin/editor authentication)
  created_by?: number;    // Filter by creator user ID
  updated_by?: number;    // Filter by last updater user ID
  workflow_status?: string; // Filter by workflow state (draft, pending_review, approved, rejected, scheduled, published, archived)
  created_after?: string; // ISO 8601 timestamp (e.g., "2024-01-01T00:00:00Z")
  created_before?: string; // ISO 8601 timestamp
  updated_after?: string; // ISO 8601 timestamp
  updated_before?: string; // ISO 8601 timestamp
  published_after?: string; // ISO 8601 timestamp
  published_before?: string; // ISO 8601 timestamp
}
```

**Admin Request Example:**
```http
GET /api/v1/reports?created_by=5&workflow_status=pending_review&created_after=2024-01-01T00:00:00Z
Authorization: Bearer {access_token}
```

### Request Example
```http
GET /api/v1/reports?page=1&limit=10&status=published&category=Pharmaceuticals
```

### Response Structure
```typescript
{
  "success": true,
  "data": [
    {
      "id": 1,
      "title": "Global Healthcare Market Analysis 2024",
      "slug": "global-healthcare-market-analysis-2024",
      "summary": "Comprehensive analysis of the global healthcare market...",
      "description": "Detailed market research report...",
      "category_id": 5,
      "category_name": "Pharmaceuticals",
      "geography": ["Global", "North America", "Europe"],
      "publish_date": "2024-01-15",
      "price": 3490,
      "discounted_price": 3090,
      "currency": "USD",
      "status": "published",
      "access_type": "paid",
      "page_count": 145,
      "formats": ["PDF", "Excel", "Word"],
      "market_metrics": {
        "currentRevenue": "$87.4 billion",
        "currentYear": 2024,
        "forecastRevenue": "$286.2 billion",
        "forecastYear": 2032,
        "cagr": "16.8%",
        "cagrStartYear": 2024,
        "cagrEndYear": 2032
      },
      "author_ids": [1, 5],
      "key_players": [
        {
          "name": "Pfizer Inc.",
          "marketShare": "15.2%",
          "rank": 1,
          "description": "Leading pharmaceutical company"
        }
      ],
      "sections": {
        "keyPlayers": "<p>Key players HTML content...</p>",
        "marketDetails": "<p>Market details HTML content...</p>",
        // ... other sections
      },
      "faqs": [
        {
          "question": "What is the market size?",
          "answer": "The market size is estimated at $87.4 billion..."
        }
      ],
      "meta_title": "Global Healthcare Market Report 2024",
      "meta_description": "Comprehensive market analysis...",
      "meta_keywords": "healthcare,market research,pharmaceuticals",
      "thumbnail_url": "https://example.com/thumbnails/report-1.jpg",
      "is_featured": true,
      "view_count": 1523,
      "download_count": 342,
      "created_at": "2024-01-10T10:30:00Z",
      "updated_at": "2024-01-15T14:22:00Z",

      // Admin-only fields (included when authenticated as admin/editor)
      "created_by": 5,
      "updated_by": 5,
      "internal_notes": "Pending final review from marketing team",
      "workflow_status": "approved",
      "scheduled_publish_at": null,
      "approved_by": 1,
      "approved_at": "2024-01-14T16:30:00Z"
    }
    // ... more reports
  ],
  "meta": {
    "page": 1,
    "limit": 10,
    "total": 157,
    "total_pages": 16
  }
}
```

### Error Response
```typescript
{
  "success": false,
  "error": "Invalid filter parameters"
}
```

---

## 2. Get Report by Slug

### Endpoint
```
GET /api/v1/reports/{slug}
```

### Authentication
❌ Not required (public endpoint)

### Path Parameters
- `slug` (string): URL-friendly report identifier

### Request Example
```http
GET /api/v1/reports/global-healthcare-market-analysis-2024
```

### Response Structure
```typescript
{
  "success": true,
  "data": {
    // All fields from ApiReport (same as list response)
    "id": 1,
    "title": "Global Healthcare Market Analysis 2024",
    "slug": "global-healthcare-market-analysis-2024",
    // ... (all standard report fields)

    // Additional relation fields
    "category_name": "Pharmaceuticals",
    "author": {
      "id": 1,
      "email": "researcher@example.com",
      "name": "Dr. John Smith"
    },
    "charts": [
      {
        "id": 101,
        "report_id": 1,
        "title": "Market Growth by Region",
        "description": "Regional market analysis",
        "chart_type": "bar",
        "data_points": 7,
        "order": 1,
        "is_active": true,
        "created_at": "2024-01-10T10:30:00Z",
        "updated_at": "2024-01-15T14:22:00Z"
      }
    ],
    "versions": [
      {
        "id": 1,
        "report_id": 1,
        "version_number": 1,
        "published_at": "2024-01-15T00:00:00Z",
        "published_by": 1,
        "sections": { /* snapshot of sections */ },
        "meta_title": "Global Healthcare Market Report 2024",
        "meta_description": "Comprehensive market analysis...",
        "meta_keywords": "healthcare,market research,pharmaceuticals",
        "created_at": "2024-01-15T10:30:00Z"
      }
    ]
  }
}
```

---

## 3. Create New Report

### Endpoint
```
POST /api/v1/reports
```

### Authentication
✅ Required (admin/editor only)

### Request Headers
```http
Authorization: Bearer {access_token}
Content-Type: application/json
```

### Request Body
```typescript
{
  "title": "New Market Research Report 2024",
  "summary": "Brief overview of the report (minimum 50 characters)",
  "category_id": 5,
  "geography": ["Global", "North America"],
  "price": 3490,
  "discounted_price": 3090,
  "access_type": "paid",
  "status": "draft",  // "draft" or "published"
  "page_count": 120,
  "formats": ["PDF", "Excel"],
  "market_metrics": {
    "currentRevenue": "$100 billion",
    "currentYear": 2024,
    "forecastRevenue": "$300 billion",
    "forecastYear": 2032,
    "cagr": "15.5%",
    "cagrStartYear": 2024,
    "cagrEndYear": 2032
  },
  "author_ids": [1],
  "key_players": [
    {
      "name": "Company A",
      "marketShare": "20%",
      "rank": 1
    }
  ],
  "sections": {
    "keyPlayers": "<p>Key players analysis...</p>",
    "marketDetails": "<p>Detailed market info...</p>",
    "tableOfContents": "<p>Table of contents...</p>"
  },
  "faqs": [
    {
      "question": "What is the scope of this report?",
      "answer": "This report covers global market trends..."
    }
  ],
  "meta_title": "New Market Research Report",
  "meta_description": "Comprehensive market analysis",
  "meta_keywords": "market,research,analysis"
}
```

### Minimum Required Fields (for Draft)
```typescript
{
  "title": "Untitled Report (Draft)",
  "summary": "Draft in progress",
  "category_id": 1,
  "geography": ["Global"],
  "price": 0,
  "discounted_price": 0,
  "access_type": "free",
  "status": "draft",
  "sections": {
    "keyPlayers": "",
    "marketDetails": "",
    "tableOfContents": ""
  },
  "meta_title": "",
  "meta_description": "",
  "meta_keywords": ""
}
```

### Success Response
```typescript
{
  "success": true,
  "data": {
    "id": 158,
    "title": "New Market Research Report 2024",
    "slug": "new-market-research-report-2024",  // Auto-generated
    // ... all report fields including relations
    "created_at": "2024-01-20T10:30:00Z",
    "updated_at": "2024-01-20T10:30:00Z"
  }
}
```

### Error Responses
```typescript
// Validation Error
{
  "success": false,
  "error": "Title must be at least 10 characters"
}

// Authentication Error
{
  "success": false,
  "error": "No authentication token"
}

// Authorization Error
{
  "success": false,
  "error": "Insufficient permissions"
}
```

---

## 4. Update Existing Report

### Endpoint
```
PUT /api/v1/reports/{id}
```

### Authentication
✅ Required (admin/editor only)

### Path Parameters
- `id` (number): Report ID

### Request Headers
```http
Authorization: Bearer {access_token}
Content-Type: application/json
```

### Request Body
All fields are optional (partial update supported):

```typescript
{
  "title"?: "Updated Report Title",
  "summary"?: "Updated summary",
  "status"?: "published",  // Change from draft to published
  "price"?: 3990,
  "sections"?: {
    "marketDetails": "<p>Updated market details...</p>"
    // Other sections unchanged
  },
  "meta_title"?: "Updated Meta Title",
  "meta_description"?: "Updated meta description",
  "meta_keywords"?: "keyword1,keyword2,keyword3"
  // ... any other fields to update
}
```

### Success Response
```typescript
{
  "success": true,
  "data": {
    "id": 1,
    "title": "Updated Report Title",
    "slug": "updated-report-title",  // Auto-updated if title changed
    // ... all report fields with updates applied
    "updated_at": "2024-01-20T15:45:00Z"
  }
}
```

### Error Responses
Same as Create endpoint

---

## 5. Delete Report

### Endpoint
```
DELETE /api/v1/reports/{id}
```

### Authentication
✅ Required (admin only)

### Path Parameters
- `id` (number): Report ID

### Request Headers
```http
Authorization: Bearer {access_token}
```

### Request Example
```http
DELETE /api/v1/reports/158
```

### Success Response
```typescript
{
  "success": true,
  "data": {
    "message": "Report deleted successfully"
  }
}
```

### Error Responses
```typescript
// Report Not Found
{
  "success": false,
  "error": "Report not found"
}

// Authorization Error
{
  "success": false,
  "error": "Only admins can delete reports"
}
```

---

## 6. Search Reports

### Endpoint
```
GET /api/v1/search
```

### Authentication
❌ Not required (public endpoint)

### Query Parameters
```typescript
{
  q: string;        // Search query (required)
  page?: number;    // Page number (default: 1)
  limit?: number;   // Items per page (default: 10)
}
```

### Request Example
```http
GET /api/v1/search?q=healthcare&page=1&limit=20
```

### Response Structure
Same as "Get All Reports" response

---

## 7. Get Reports by Category

### Endpoint
```
GET /api/v1/categories/{slug}/reports
```

### Authentication
❌ Not required (public endpoint)

### Path Parameters
- `slug` (string): Category slug

### Query Parameters
```typescript
{
  page?: number;    // Page number (default: 1)
  limit?: number;   // Items per page (default: 10)
}
```

### Request Example
```http
GET /api/v1/categories/pharmaceuticals/reports?page=1&limit=10
```

### Response Structure
Same as "Get All Reports" response

---

## Data Type Specifications

### Report Sections
All sections support HTML content:
- `keyPlayers` - Optional
- `marketDetails` - Min 100 chars when publishing
- `tableOfContents` - Min 50 chars when publishing

### Market Metrics
All fields optional:
- `currentRevenue` - String (e.g., "$87.4 billion")
- `currentYear` - Number (e.g., 2024)
- `forecastRevenue` - String (e.g., "$286.2 billion")
- `forecastYear` - Number (e.g., 2032)
- `cagr` - String (e.g., "16.8%")
- `cagrStartYear` - Number (e.g., 2024)
- `cagrEndYear` - Number (e.g., 2032)

### Key Player
- `name` - String (required, min 2 chars)
- `marketShare` - String (optional, e.g., "15.2%")
- `rank` - Number (optional)
- `description` - String (optional)

### FAQ
- `question` - String (required, min 5 chars)
- `answer` - String (required, min 10 chars)

### SEO Metadata
All fields optional:
- `meta_title` - String (max 255 chars)
- `meta_description` - String (max 500 chars)
- `meta_keywords` - String (comma-separated keywords, max 500 chars)

---

## Status Codes

| Code | Description |
|------|-------------|
| 200 | Success |
| 201 | Created (POST successful) |
| 400 | Bad Request (validation error) |
| 401 | Unauthorized (no/invalid token) |
| 403 | Forbidden (insufficient permissions) |
| 404 | Not Found (report doesn't exist) |
| 500 | Internal Server Error |

---

## Authentication Flow

### Token Included in Headers
```http
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

### Auto Token Refresh
The API client automatically:
1. Checks token expiration before each request
2. Calls `/api/v1/auth/refresh` if expired
3. Retries original request with new token
4. Clears tokens and throws error if refresh fails

---

## Frontend Usage Examples

### Fetch All Published Reports
```typescript
import { fetchReports } from '@/lib/api/reports.api';

const response = await fetchReports({
  status: 'published',
  page: 1,
  limit: 10
});

console.log(response.data);      // Array of reports
console.log(response.meta.total); // Total count
```

### Create New Draft Report
```typescript
import { createReport } from '@/lib/api/reports.api';

const response = await createReport({
  title: "New Report",
  summary: "Brief overview of at least 50 characters...",
  category_id: 5,
  geography: ["Global"],
  status: "draft",
  // ... other fields
});

console.log(response.data.id);    // New report ID
console.log(response.data.slug);  // Auto-generated slug
```

### Update Report Status
```typescript
import { updateReport } from '@/lib/api/reports.api';

const response = await updateReport(158, {
  status: "published"
});

console.log(response.data.status); // "published"
```

### Delete Report
```typescript
import { deleteReport } from '@/lib/api/reports.api';

const response = await deleteReport(158);
console.log(response.data.message); // "Report deleted successfully"
```

---

## Notes

1. **Snake Case vs Camel Case**:
   - API uses `snake_case` (e.g., `category_id`, `access_type`)
   - Frontend types may use `camelCase` (conversion handled by wrapper)

2. **Partial Updates**:
   - PUT endpoint supports partial updates
   - Only send fields you want to change
   - Missing fields keep existing values

3. **Slug Generation**:
   - Auto-generated from title
   - Updated when title changes
   - Must be unique across all reports

4. **Version History**:
   - Automatically created on publish
   - Snapshots sections and SEO metadata fields
   - Cannot be modified after creation

5. **Draft Validation**:
   - Drafts can be saved with minimal data
   - Publishing requires all required fields
   - Frontend validation is stricter than API

---

## File References

- **API Types**: `lib/types/api-types.ts`
- **API Service**: `lib/api/reports.api.ts`
- **API Client**: `lib/api/client.ts`
- **Frontend Types**: `lib/types/reports.ts`
- **Hooks**: `hooks/use-report.ts`, `hooks/use-reports.ts`
