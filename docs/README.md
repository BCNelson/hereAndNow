# Here and Now API Documentation

This directory contains comprehensive documentation for the Here and Now task management API.

## Files

- **[api.html](./api.html)** - Interactive HTML documentation with examples, schemas, and endpoint details
- **[../specs/001-build-an-application/contracts/api-v1.yaml](../specs/001-build-an-application/contracts/api-v1.yaml)** - OpenAPI 3.0 specification with comprehensive examples

## Viewing Documentation

### HTML Documentation
Open `api.html` in any web browser for an interactive documentation experience featuring:
- Comprehensive endpoint documentation with examples
- Request/response schemas with validation rules
- Context-aware filtering explanations
- Mobile-responsive design

### OpenAPI Specification
The `api-v1.yaml` file contains the complete OpenAPI 3.0 specification with:
- Detailed request/response examples
- Comprehensive schema definitions with validation constraints
- Authentication flows
- Error response documentation

## Key API Features

### Context-Aware Filtering
The API automatically filters tasks based on:
- **Location**: Tasks only appear when you're within the required location radius
- **Time**: Tasks are filtered by available time vs. estimated completion time  
- **Energy Level**: High-energy tasks are hidden when energy is low
- **Calendar Integration**: Tasks are filtered based on calendar conflicts

### Real-time Updates
- Server-Sent Events (SSE) at `/events` endpoint for live task list updates
- Context changes automatically trigger task list refreshes
- Real-time collaboration for shared task lists

### Audit Transparency
- Complete audit trails available at `/tasks/{taskId}/audit`
- Detailed explanations for why tasks are visible or hidden
- Filter rule execution history with timestamps

### Natural Language Processing
- Create tasks from natural language at `/tasks/natural`
- Support for text, voice, and image inputs
- Automatic location and time extraction

## Authentication

All endpoints (except `/auth/login`) require JWT authentication:

```bash
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

## Quick Start Examples

### Get Filtered Tasks
```bash
curl -H "Authorization: Bearer YOUR_TOKEN" \
  http://localhost:8080/api/v1/tasks
```

### Create a Task
```bash
curl -X POST \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Buy groceries",
    "description": "Pick up milk, bread, and eggs",
    "priority": 3,
    "estimated_minutes": 30,
    "location_ids": ["location-uuid"]
  }' \
  http://localhost:8080/api/v1/tasks
```

### Update Context
```bash
curl -X POST \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "current_latitude": 40.7128,
    "current_longitude": -74.0060,
    "available_minutes": 60,
    "energy_level": 4
  }' \
  http://localhost:8080/api/v1/context
```

## Error Handling

The API returns standard HTTP status codes:
- `200` - Success
- `201` - Created
- `400` - Bad Request (validation errors)
- `401` - Unauthorized (invalid/missing token)
- `403` - Forbidden (insufficient permissions)
- `404` - Not Found
- `500` - Internal Server Error

Error responses include detailed messages:
```json
{
  "error": "Validation failed",
  "details": {
    "title": "Title is required",
    "priority": "Priority must be between 1 and 5"
  }
}
```

## Rate Limiting

- 1000 requests per hour per user
- 100 requests per minute for context updates
- 500 requests per hour for task creation/updates

Rate limit headers are included in all responses:
- `X-RateLimit-Limit`
- `X-RateLimit-Remaining` 
- `X-RateLimit-Reset`

## Support

For questions about the API documentation:
1. Check the interactive HTML documentation for detailed examples
2. Review the OpenAPI specification for complete schema definitions
3. Consult the audit endpoints to understand filtering behavior
4. Refer to the main project documentation for deployment and setup

---

Generated from OpenAPI specification on 2025-09-09