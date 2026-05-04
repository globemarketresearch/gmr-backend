# Form Submissions API Documentation

This document provides detailed information about the Form Submissions API endpoints for managing contact forms and sample request forms.

## Table of Contents
- [Overview](#overview)
- [Endpoints](#endpoints)
- [Data Models](#data-models)
- [Examples](#examples)
- [Error Handling](#error-handling)

## Overview

The Form Submissions API allows you to:
- Submit contact forms and sample request forms
- Retrieve form submissions with filtering and pagination
- Manage submission status (pending, processed, archived)
- Get statistics about form submissions
- Delete submissions (admin only)

### Base URL
```
http://localhost:8081/api/v1/forms
```

### Authentication
- **Public endpoints**: Form submission, viewing submissions, statistics
- **Protected endpoints**: Delete, bulk delete, status updates (requires admin/editor role)

## Endpoints

### 1. Create Form Submission (Public)

Submit a new contact form or request sample form.

**Endpoint**: `POST /api/v1/forms/submissions`

**Request Body**:
```json
{
  "category": "contact",
  "data": {
    "fullName": "John Doe",
    "email": "john@example.com",
    "company": "HealthTech Inc",
    "phone": "+1234567890",
    "country": "United States",
    "subject": "Custom Research Request",
    "message": "I need information about telemedicine market..."
  },
  "metadata": {
    "submittedAt": "2026-01-11T10:30:00Z",
    "referrer": "/reports/telemedicine-market"
  }
}
```

**Response** (201 Created):
```json
{
  "success": true,
  "submissionId": "sub_a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "category": "contact",
  "message": "Form submitted successfully",
  "createdAt": "2026-01-11T10:30:00Z"
}
```

**Category Types**:
- `contact` - Contact form submission
- `request-sample` - Sample request form submission

**Contact Form Fields** (all required except phone and country):
- `fullName` (string, required)
- `email` (string, required)
- `company` (string, required)
- `phone` (string, optional)
- `country` (string, optional)
- `subject` (string, required)
- `message` (string, required)

**Request Sample Form Fields** (all required except phone, country, and additionalInfo):
- `fullName` (string, required)
- `email` (string, required)
- `company` (string, required)
- `jobTitle` (string, required)
- `phone` (string, optional)
- `country` (string, optional)
- `reportTitle` (string, required)
- `additionalInfo` (string, optional)

---

### 2. Get All Submissions (Public)

Retrieve a paginated list of form submissions with optional filtering.

**Endpoint**: `GET /api/v1/forms/submissions`

**Query Parameters**:
- `category` (optional) - Filter by category: `contact`, `request-sample`
- `status` (optional) - Filter by status: `pending`, `processed`, `archived`
- `dateFrom` (optional) - Start date (ISO 8601 format)
- `dateTo` (optional) - End date (ISO 8601 format)
- `search` (optional) - Search in name, email, company
- `page` (optional, default: 1) - Page number
- `limit` (optional, default: 20, max: 100) - Items per page
- `sortBy` (optional, default: createdAt) - Sort field: `createdAt`, `company`, `name`
- `sortOrder` (optional, default: desc) - Sort order: `asc`, `desc`

**Example Request**:
```
GET /api/v1/forms/submissions?category=contact&status=pending&page=1&limit=20
```

**Response** (200 OK):
```json
{
  "success": true,
  "data": [
    {
      "id": "sub_a1b2c3d4-e5f6-7890-abcd-ef1234567890",
      "category": "contact",
      "status": "pending",
      "data": {
        "fullName": "John Doe",
        "email": "john@example.com",
        "company": "HealthTech Inc",
        "phone": "+1234567890",
        "country": "United States",
        "subject": "Custom Research Request",
        "message": "I need information about..."
      },
      "metadata": {
        "submittedAt": "2026-01-11T10:30:00Z",
        "ipAddress": "192.168.1.1",
        "userAgent": "Mozilla/5.0...",
        "referrer": "/reports/telemedicine-market"
      },
      "createdAt": "2026-01-11T10:30:00Z",
      "updatedAt": "2026-01-11T10:30:00Z"
    }
  ],
  "meta": {
    "page": 1,
    "limit": 20,
    "total": 156,
    "total_pages": 8
  }
}
```

---

### 3. Get Single Submission (Public)

Retrieve a specific form submission by ID.

**Endpoint**: `GET /api/v1/forms/submissions/:id`

**Example Request**:
```
GET /api/v1/forms/submissions/sub_a1b2c3d4-e5f6-7890-abcd-ef1234567890
```

**Response** (200 OK):
```json
{
  "success": true,
  "data": {
    "id": "sub_a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "category": "contact",
    "status": "pending",
    "data": {
      "fullName": "John Doe",
      "email": "john@example.com",
      "company": "HealthTech Inc",
      "phone": "+1234567890",
      "country": "United States",
      "subject": "Custom Research Request",
      "message": "I need information about..."
    },
    "metadata": {
      "submittedAt": "2026-01-11T10:30:00Z",
      "ipAddress": "192.168.1.1",
      "userAgent": "Mozilla/5.0...",
      "referrer": "/reports/telemedicine-market"
    },
    "createdAt": "2026-01-11T10:30:00Z",
    "updatedAt": "2026-01-11T10:30:00Z"
  }
}
```

---

### 4. Get Submissions by Category (Public)

Retrieve form submissions filtered by category.

**Endpoint**: `GET /api/v1/forms/submissions/category/:category`

**Example Request**:
```
GET /api/v1/forms/submissions/category/contact?page=1&limit=20
```

**Response**: Same structure as "Get All Submissions"

---

### 5. Get Statistics (Public)

Get statistics about form submissions.

**Endpoint**: `GET /api/v1/forms/stats`

**Response** (200 OK):
```json
{
  "success": true,
  "data": {
    "total": 730,
    "byCategory": {
      "contact": 450,
      "request-sample": 280
    },
    "byStatus": {
      "pending": 85,
      "processed": 595,
      "archived": 50
    },
    "recent": {
      "today": 12,
      "thisWeek": 64,
      "thisMonth": 248
    }
  }
}
```

---

### 6. Delete Submission (Admin Only)

Delete a single form submission.

**Endpoint**: `DELETE /api/v1/forms/submissions/:id`

**Headers**:
```
Authorization: Bearer <admin_token>
```

**Example Request**:
```
DELETE /api/v1/forms/submissions/sub_a1b2c3d4-e5f6-7890-abcd-ef1234567890
```

**Response** (200 OK):
```json
{
  "success": true,
  "data": {
    "success": true,
    "message": "Submission deleted successfully",
    "deletedId": "sub_a1b2c3d4-e5f6-7890-abcd-ef1234567890"
  }
}
```

---

### 7. Bulk Delete Submissions (Admin Only)

Delete multiple form submissions at once.

**Endpoint**: `DELETE /api/v1/forms/submissions`

**Headers**:
```
Authorization: Bearer <admin_token>
```

**Request Body**:
```json
{
  "ids": [
    "sub_a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "sub_b2c3d4e5-f6a7-8901-bcde-f12345678901"
  ]
}
```

**Response** (200 OK):
```json
{
  "success": true,
  "data": {
    "success": true,
    "message": "2 submissions deleted successfully",
    "deletedCount": 2,
    "deletedIds": [
      "sub_a1b2c3d4-e5f6-7890-abcd-ef1234567890",
      "sub_b2c3d4e5-f6a7-8901-bcde-f12345678901"
    ]
  }
}
```

---

### 8. Update Submission Status (Admin/Editor Only)

Update the processing status of a submission.

**Endpoint**: `PATCH /api/v1/forms/submissions/:id/status`

**Headers**:
```
Authorization: Bearer <admin_or_editor_token>
```

**Request Body**:
```json
{
  "status": "processed"
}
```

**Valid Status Values**:
- `pending` - New submission, not yet processed
- `processed` - Submission has been handled
- `archived` - Submission is archived

**Response** (200 OK):
```json
{
  "success": true,
  "data": {
    "message": "Status updated successfully",
    "status": "processed"
  }
}
```

---

## Data Models

### FormSubmission
```typescript
{
  id: string;                    // Primary key (e.g., "sub_" + uuid)
  category: 'contact' | 'request-sample';
  status: 'pending' | 'processed' | 'archived';
  data: FormData;                // Form-specific fields (JSONB)
  metadata: {
    submittedAt: string;         // ISO timestamp
    ipAddress?: string;
    userAgent?: string;
    referrer?: string;
  };
  processedAt?: string;          // When marked as processed
  processedBy?: number;          // Admin user ID
  notes?: string;                // Admin notes
  createdAt: string;
  updatedAt: string;
}
```

---

## Examples

### Example 1: Submit Contact Form from Frontend

```javascript
async function submitContactForm(formData) {
  const response = await fetch('http://localhost:8081/api/v1/forms/submissions', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      category: 'contact',
      data: {
        fullName: formData.fullName,
        email: formData.email,
        company: formData.company,
        phone: formData.phone,
        country: formData.country,
        subject: formData.subject,
        message: formData.message,
      },
      metadata: {
        submittedAt: new Date().toISOString(),
        referrer: window.location.pathname,
      }
    })
  });

  return await response.json();
}
```

### Example 2: Submit Request Sample Form

```javascript
async function submitRequestSampleForm(formData) {
  const response = await fetch('http://localhost:8081/api/v1/forms/submissions', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      category: 'request-sample',
      data: {
        fullName: formData.fullName,
        email: formData.email,
        company: formData.company,
        jobTitle: formData.jobTitle,
        phone: formData.phone,
        country: formData.country,
        reportTitle: formData.reportTitle,
        additionalInfo: formData.additionalInfo,
      },
      metadata: {
        submittedAt: new Date().toISOString(),
        referrer: window.location.pathname,
      }
    })
  });

  return await response.json();
}
```

### Example 3: Fetch Pending Submissions (Admin)

```javascript
async function fetchPendingSubmissions(token) {
  const response = await fetch(
    'http://localhost:8081/api/v1/forms/submissions?status=pending&page=1&limit=20',
    {
      headers: {
        'Authorization': `Bearer ${token}`
      }
    }
  );

  return await response.json();
}
```

### Example 4: Update Submission Status (Admin)

```javascript
async function markAsProcessed(submissionId, token) {
  const response = await fetch(
    `http://localhost:8081/api/v1/forms/submissions/${submissionId}/status`,
    {
      method: 'PATCH',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${token}`
      },
      body: JSON.stringify({
        status: 'processed'
      })
    }
  );

  return await response.json();
}
```

---

## Error Handling

### Error Response Format
```json
{
  "success": false,
  "error": "Error message here"
}
```

### Common Error Codes

| Status Code | Description |
|-------------|-------------|
| 400 | Bad Request - Invalid input or validation error |
| 401 | Unauthorized - Authentication required |
| 403 | Forbidden - Insufficient permissions |
| 404 | Not Found - Submission not found |
| 500 | Internal Server Error - Server-side error |

### Example Error Responses

**400 Bad Request** (Missing required field):
```json
{
  "success": false,
  "error": "fullName is required"
}
```

**400 Bad Request** (Invalid category):
```json
{
  "success": false,
  "error": "invalid category: must be 'contact' or 'request-sample'"
}
```

**401 Unauthorized**:
```json
{
  "success": false,
  "error": "Missing or invalid authentication token"
}
```

**404 Not Found**:
```json
{
  "success": false,
  "error": "Submission not found"
}
```

---

## Testing with cURL

### Submit a Contact Form
```bash
curl -X POST http://localhost:8081/api/v1/forms/submissions \
  -H "Content-Type: application/json" \
  -d '{
    "category": "contact",
    "data": {
      "fullName": "John Doe",
      "email": "john@example.com",
      "company": "HealthTech Inc",
      "phone": "+1234567890",
      "country": "United States",
      "subject": "Custom Research Request",
      "message": "I need information about telemedicine market"
    }
  }'
```

### Get All Submissions
```bash
curl http://localhost:8081/api/v1/forms/submissions?page=1&limit=20
```

### Get Statistics
```bash
curl http://localhost:8081/api/v1/forms/stats
```

### Delete a Submission (Admin)
```bash
curl -X DELETE http://localhost:8081/api/v1/forms/submissions/sub_a1b2c3d4-e5f6-7890-abcd-ef1234567890 \
  -H "Authorization: Bearer YOUR_ADMIN_TOKEN"
```

---

## Database Schema

The `form_submissions` table structure:

```sql
CREATE TABLE form_submissions (
  id VARCHAR(50) PRIMARY KEY,
  category VARCHAR(20) NOT NULL,
  status VARCHAR(20) DEFAULT 'pending',
  data JSONB NOT NULL,
  metadata JSONB,
  processed_at TIMESTAMP,
  processed_by INTEGER,
  notes TEXT,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexes
CREATE INDEX idx_form_submissions_category ON form_submissions(category);
CREATE INDEX idx_form_submissions_status ON form_submissions(status);
```

---

## Notes

1. **IP Address and User Agent**: These are automatically captured from the request if not provided in the metadata.

2. **Caching**: List endpoints are cached for 5 minutes for non-filtered queries. Statistics are cached for 2 minutes.

3. **Search**: The search functionality searches across `fullName`, `email`, and `company` fields in the JSONB data.

4. **Sorting**: You can sort by `createdAt`, `company`, or `name` (fullName).

5. **Admin Access**: Delete and status update operations require admin or editor authentication.

6. **Validation**: The API validates required fields based on the form category (contact vs request-sample).

---

## Integration Checklist

- [ ] Update frontend forms to use the new API endpoints
- [ ] Test form submission from both Contact and Request Sample pages
- [ ] Verify email notifications (if configured)
- [ ] Set up admin dashboard to view and manage submissions
- [ ] Configure rate limiting for form submissions (if needed)
- [ ] Test error handling and validation
- [ ] Monitor submission statistics

---

For more information or support, please refer to the main API documentation or contact the development team.
