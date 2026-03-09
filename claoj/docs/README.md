# CLAOJ API v2 Documentation

This directory contains Swagger/OpenAPI documentation for the CLAOJ API v2.

## Generating Documentation

The API documentation is generated using [swaggo/swag](https://github.com/swaggo/swag) from Go comments in the source code.

### Installation

```bash
go install github.com/swaggo/swag/cmd/swag@latest
```

### Generate Docs

From the `claoj-go` directory:

```bash
swag init --parseDependency --parseInternal
```

This will generate:
- `docs/docs.go` - Main documentation file
- `docs/swagger.json` - OpenAPI 2.0 JSON specification
- `docs/swagger.yaml` - OpenAPI 2.0 YAML specification

### View Documentation

Once generated, you can view the documentation by:

1. **Swagger UI**: Access at `http://localhost:8080/swagger/index.html` (when Swagger UI middleware is enabled)

2. **Raw JSON**: Access at `http://localhost:8080/swagger/doc.json`

## Adding API Documentation

To document a new API endpoint, add comments in the following format:

```go
// EndpointName – GET /api/v2/endpoint
// @Description  Human-readable description of what this endpoint does
// @Tags         Category
// @Summary      Short summary
// @Produce      json
// @Param        param1  query     int     false  "Description"  default(100)
// @Param        param2  path      string  true   "Required param"
// @Success      200     {object}  ReturnType
// @Failure      400     {object}  ErrorType
// @Router       /endpoint [get]
func EndpointName(c *gin.Context) {
    // ...
}
```

### Common Annotations

| Annotation | Description |
|------------|-------------|
| `@Description` | Detailed description of the endpoint |
| `@Tags` | Category/grouping for the endpoint |
| `@Summary` | Short summary (shown in Swagger UI) |
| `@Produce` | Response content type (json, xml, etc.) |
| `@Accept` | Request content type |
| `@Param` | Parameter definition |
| `@Success` | Success response |
| `@Failure` | Error response |
| `@Security` | Authentication requirement |
| `@Router` | HTTP method and path |

### Parameter Types

```go
// @Param id path int true "Resource ID"
// @Param name query string false "Filter by name"
// @Param page query int false "Page number" default(1)
// @Param body body CreateRequest true "Request body"
```

### Response Types

```go
// @Success 200 {object} map[string]interface{}
// @Success 200 {object} User
// @Failure 400 {object} ErrorResponse
// @Failure 404 {string} string "Not found"
```

## Documented Endpoints

### Authentication
- POST `/auth/login` - User login
- POST `/auth/logout` - User logout
- POST `/auth/register` - User registration
- POST `/auth/refresh` - Refresh access token

### Users
- GET `/users` - List users
- GET `/user/:user` - Get user profile
- GET `/user/:user/solved` - Get user's solved problems
- GET `/user/:user/rating` - Get user's rating history
- GET `/user/:user/pp-breakdown` - Get user's PP breakdown
- GET `/user/:user/analytics` - Get user analytics

### Problems
- GET `/problems` - List problems
- GET `/problem/:code` - Get problem details
- POST `/problem/:code/submit` - Submit solution

### Contests
- GET `/contests` - List contests
- GET `/contest/:key` - Get contest details
- GET `/contest/:key/ranking` - Get contest ranking
- POST `/contest/:key/join` - Join contest

### Submissions
- GET `/submissions` - List submissions
- GET `/submission/:id` - Get submission details

### Admin (requires admin authentication)
- CRUD operations for users, problems, contests
- Submission rejudge and rescore
- Role and permission management

## Security

### Authentication Methods

1. **API Token** (64 hex characters)
   ```
   Authorization: <token>
   ```

2. **JWT Bearer Token**
   ```
   Authorization: Bearer <jwt_token>
   ```

3. **Access Token Cookie**
   ```
   Cookie: access_token=<jwt_token>
   ```

### Rate Limiting

All API endpoints are rate-limited. See rate limit headers in responses:
- `X-RateLimit-Limit` - Maximum requests per window
- `X-RateLimit-Remaining` - Remaining requests
- `X-RateLimit-Reset` - Window reset time

## Example Requests

### Get User Profile

```bash
curl -X GET "https://beta.claoj.edu.vn/api/v2/user/example" \
  -H "Accept: application/json"
```

### Submit Solution

```bash
curl -X POST "https://beta.claoj.edu.vn/api/v2/problem/EXAMPLE/submit" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d '{
    "language_id": 54,
    "source": "#include <iostream>\nint main() { return 0; }"
  }'
```

### Join Contest

```bash
curl -X POST "https://beta.claoj.edu.vn/api/v2/contest/EXAMPLE/join" \
  -H "Authorization: Bearer <token>" \
  -d '{}'
```

## API Versioning

The current API version is **v2**. The legacy Django API (v1) is deprecated and will be removed in a future release.

Base path for all v2 endpoints: `/api/v2`

## Error Responses

All errors follow this format:

```json
{
  "error": "error_code_or_message"
}
```

Common error codes:
- `unauthorized` - Authentication required
- `forbidden` - Insufficient permissions
- `not_found` - Resource not found
- `invalid_request` - Malformed request
- `rate_limited` - Too many requests

## Contact

For API issues or questions:
- GitHub Issues: https://github.com/CLAOJ/claoj/issues
- Email: support@claoj.edu.vn
