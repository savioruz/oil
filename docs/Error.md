# API Error Keys Documentation

## Overview

All API error responses return a single `error` field containing a machine-readable error key. This allows frontends to:
- Implement consistent error handling across all endpoints
- Provide translated error messages based on user locale
- Display appropriate UI for specific error types
- Log and track error patterns

## Response Format

All error responses follow a simple, consistent format with a single `error` field containing the machine-readable error key:

```json
{
  "error": "error.key.here"
}
```

The HTTP status code provides the error category (4xx for client errors, 5xx for server errors), and the error key provides specific details about what went wrong.

### Examples

**Validation Error (Field-Specific):**
```http
POST /api/galleries
422 Unprocessable Entity

{
  "error": "validation.required.title"
}
```

**Validation Error (Invalid URL):**
```http
POST /api/galleries
422 Unprocessable Entity

{
  "error": "validation.url.images"
}
```

**Resource Not Found:**
```http
GET /api/galleries/abc123
404 Not Found

{
  "error": "gallery.not_found"
}
```

**Authentication Error:**
```http
GET /api/galleries
401 Unauthorized

{
  "error": "auth.unauthorized"
}
```

**Server Error:**
```http
GET /api/galleries
500 Internal Server Error

{
  "error": "database.query_failed"
}
```

## Error Key Categories

Error keys follow the pattern: `category.specific_error`

- **validation**: Input validation errors (400, 422)
- **auth**: Authentication and authorization errors (401, 403)
- **resource**: Resource-related errors (404, 409)
- **database**: Database operation errors (500)
- **server**: General server errors (500, 503)
- **external**: External service errors (500, 503)
- **gallery**: Gallery module-specific errors (404, 500)
- **todo**: Todo module-specific errors (404, 500)

## Complete Error Key Reference

### Validation Errors (400, 422)

**Field-Specific Validation Keys:**

Validation errors return field-specific error keys in the format: `validation.{rule}.{field_name}`

Examples:
- `validation.required.title` - Title field is required
- `validation.min.title` - Title doesn't meet minimum length
- `validation.url.images` - Images field contains invalid URL
- `validation.email.email` - Email field has invalid format

| Error Key Pattern | HTTP Status | Description | Example Response |
|-------------------|-------------|-------------|------------------|
| `validation.required.{field}` | 422 | Required field is missing | `{"error": "validation.required.title"}` |
| `validation.min.{field}` | 422 | Value too short/small | `{"error": "validation.min.title"}` |
| `validation.max.{field}` | 422 | Value too long/large | `{"error": "validation.max.description"}` |
| `validation.gte.{field}` | 422 | Value must be ≥ param | `{"error": "validation.gte.age"}` |
| `validation.lte.{field}` | 422 | Value must be ≤ param | `{"error": "validation.lte.price"}` |
| `validation.url.{field}` | 422 | URL format is invalid | `{"error": "validation.url.images"}` |
| `validation.email.{field}` | 422 | Email format is invalid | `{"error": "validation.email.email"}` |
| `validation.oneof.{field}` | 422 | Value not in allowed list | `{"error": "validation.oneof.status"}` |
| `validation.mimetypes.{field}` | 422 | File type not allowed | `{"error": "validation.mimetypes.image"}` |
| `validation.maxfilesize.{field}` | 422 | File size exceeds limit | `{"error": "validation.maxfilesize.image"}` |
| `validation.empty.{field}` | 422 | Field must be empty | `{"error": "validation.empty.optional_field"}` |
| `validation.dive.{field}` | 422 | Array element validation failed | `{"error": "validation.dive.tags"}` |

**General Validation Keys:**

| Error Key | HTTP Status | Description | Example Scenario |
|-----------|-------------|-------------|------------------|
| `validation.failed` | 422 | General validation failure | Fallback for unmapped validation errors |
| `validation.invalid_page` | 400 | Invalid page parameter | Page number is negative or not a number |
| `validation.invalid_limit` | 400 | Invalid limit parameter | Limit exceeds maximum or is not a number |

### Authentication & Authorization Errors (401, 403)

| Error Key | HTTP Status | Description | Example Scenario |
|-----------|-------------|-------------|------------------|
| `auth.unauthorized` | 401 | User not authenticated | JWT token missing or invalid |
| `auth.forbidden` | 403 | User lacks permissions | User not allowed to access resource |
| `auth.invalid_credentials` | 401 | Login credentials incorrect | Wrong username or password |
| `auth.token_expired` | 401 | Authentication token expired | JWT token past expiration time |
| `auth.token_invalid` | 401 | Token signature/format invalid | JWT token tampered or malformed |
| `auth.resource_restricted` | 403 | Resource access restricted | User not owner of resource |
| `auth.insufficient_permission` | 403 | User role lacks permission | User is viewer, needs editor role |

### Resource Errors (404, 409)

| Error Key | HTTP Status | Description | Example Scenario |
|-----------|-------------|-------------|------------------|
| `resource.not_found` | 404 | Generic resource not found | Requested ID doesn't exist |
| `resource.already_exists` | 409 | Resource with ID already exists | Duplicate key constraint |
| `resource.conflict` | 409 | Resource state conflict | Editing outdated version |
| `resource.locked` | 409 | Resource is locked | Another user editing resource |

### Database Errors (500)

| Error Key | HTTP Status | Description | Example Scenario |
|-----------|-------------|-------------|------------------|
| `database.query_failed` | 500 | Database query failed | Syntax error in SQL query |
| `database.connection_failed` | 500 | Cannot connect to database | Database server unreachable |
| `database.transaction_failed` | 500 | Transaction rollback | Deadlock or constraint violation |
| `database.constraint_violated` | 500 | Database constraint violated | Foreign key or unique constraint |

### Server Errors (500, 503)

| Error Key | HTTP Status | Description | Example Scenario |
|-----------|-------------|-------------|------------------|
| `server.internal_error` | 500 | Unexpected server error | Null pointer, panic, unhandled error |
| `server.service_unavailable` | 503 | Service temporarily unavailable | Server overloaded or restarting |
| `server.timeout` | 504 | Request processing timeout | Query took longer than timeout |
| `server.rate_limit_exceeded` | 429 | Too many requests | Client exceeded rate limit |
| `server.maintenance_mode` | 503 | Server in maintenance mode | Planned maintenance window |
| `server.shutting_down` | 503 | Server is shutting down | Graceful shutdown in progress |

### External Service Errors (500, 503)

| Error Key | HTTP Status | Description | Example Scenario |
|-----------|-------------|-------------|------------------|
| `external.service_error` | 500 | External service failure | Third-party API unavailable |
| `external.s3_upload_failed` | 500 | Failed to upload to S3 | S3 bucket permission denied |
| `external.s3_delete_failed` | 500 | Failed to delete from S3 | S3 object doesn't exist |
| `external.cache_operation_failed` | 500 | Cache operation failed | Redis connection lost |

### Gallery Module Errors

| Error Key | HTTP Status | Description | Example Scenario |
|-----------|-------------|-------------|------------------|
| `gallery.not_found` | 404 | Gallery not found | Gallery ID doesn't exist |
| `gallery.create_failed` | 500 | Failed to create gallery | Database insert error |
| `gallery.update_failed` | 500 | Failed to update gallery | Database update error |
| `gallery.delete_failed` | 500 | Failed to delete gallery | Database delete error |
| `gallery.list_failed` | 500 | Failed to list galleries | Database query error |
| `gallery.image_upload_failed` | 500 | Failed to upload image | S3 upload error |
| `gallery.image_delete_failed` | 500 | Failed to delete image | S3 delete error |
| `gallery.image_processing_failed` | 500 | Image processing failed | Thumbnail generation error |

### Todo Module Errors

| Error Key | HTTP Status | Description | Example Scenario |
|-----------|-------------|-------------|------------------|
| `todo.not_found` | 404 | Todo not found | Todo ID doesn't exist |
| `todo.create_failed` | 500 | Failed to create todo | Database insert error |
| `todo.update_failed` | 500 | Failed to update todo | Database update error |
| `todo.delete_failed` | 500 | Failed to delete todo | Database delete error |
| `todo.list_failed` | 500 | Failed to list todos | Database query error |

## Frontend Implementation Examples

### TypeScript/React

**Basic Error Handling:**

```typescript
// Error message mapping
const ERROR_MESSAGES: Record<string, string> = {
  // Validation errors - general
  'validation.failed': 'Please check your input and try again.',
  'validation.invalid_page': 'Invalid page number.',
  'validation.invalid_limit': 'Invalid limit parameter.',
  
  // Validation errors - field-specific (can use patterns or specific fields)
  'validation.required.title': 'Title is required.',
  'validation.required.email': 'Email is required.',
  'validation.min.title': 'Title must be at least 3 characters.',
  'validation.max.description': 'Description is too long.',
  'validation.url.images': 'Please provide a valid image URL.',
  'validation.email.email': 'Please provide a valid email address.',
  
  // Auth errors
  'auth.unauthorized': 'Please log in to continue.',
  'auth.forbidden': 'You don\'t have permission to perform this action.',
  
  // Resource errors
  'gallery.not_found': 'Gallery not found.',
  'todo.not_found': 'Todo item not found.',
  
  // Server errors
  'database.query_failed': 'A system error occurred. Please try again later.',
  'server.internal_error': 'An unexpected error occurred. Please try again later.',
};

// Generic fallback messages for validation patterns
function getValidationMessage(errorKey: string): string {
  // Check exact match first
  if (ERROR_MESSAGES[errorKey]) {
    return ERROR_MESSAGES[errorKey];
  }
  
  // Parse field-specific validation errors: validation.{rule}.{field}
  const match = errorKey.match(/^validation\.(\w+)\.(.+)$/);
  if (match) {
    const [, rule, field] = match;
    const fieldName = field.replace(/_/g, ' ').replace(/\[.*?\]/g, ''); // Clean field name
    
    const genericMessages: Record<string, string> = {
      required: `${fieldName} is required.`,
      min: `${fieldName} is too short.`,
      max: `${fieldName} is too long.`,
      gte: `${fieldName} must be greater than or equal to the minimum value.`,
      lte: `${fieldName} must be less than or equal to the maximum value.`,
      url: `${fieldName} must be a valid URL.`,
      email: `${fieldName} must be a valid email address.`,
      oneof: `${fieldName} has an invalid value.`,
      mimetypes: `${fieldName} has an unsupported file type.`,
      maxfilesize: `${fieldName} file size is too large.`,
    };
    
    return genericMessages[rule] || `${fieldName} validation failed.`;
  }
  
  return 'An error occurred. Please try again.';
}

// Error handler
async function handleApiCall<T>(apiCall: Promise<T>): Promise<T> {
  try {
    return await apiCall;
  } catch (error: any) {
    if (error.response?.data?.error) {
      const errorKey = error.response.data.error;
      const message = getValidationMessage(errorKey);
      
      // Show error to user
      toast.error(message);
      
      // Log for debugging
      console.error(`API Error [${errorKey}]:`, error);
      
      throw new Error(message);
    }
    throw error;
  }
}

// Usage example
try {
  await handleApiCall(createGallery(data));
  toast.success('Gallery created successfully!');
} catch (error) {
  // Error already handled by handleApiCall
}
```

**Form-Specific Error Handling:**

```typescript
interface FormErrors {
  [key: string]: string;
}

// Extract field name from validation error key
function extractFieldName(errorKey: string): string | null {
  const match = errorKey.match(/^validation\.\w+\.(.+)$/);
  return match ? match[1] : null;
}

// Handle form submission with field-specific errors
async function handleFormSubmit(data: GalleryFormData) {
  try {
    await createGallery(data);
    toast.success('Gallery created successfully!');
    router.push('/galleries');
  } catch (error: any) {
    if (error.response?.data?.error) {
      const errorKey = error.response.data.error;
      const fieldName = extractFieldName(errorKey);
      
      if (fieldName) {
        // Set field-specific error in form
        setError(fieldName, {
          type: 'server',
          message: getValidationMessage(errorKey),
        });
      } else {
        // Show general error toast
        toast.error(getValidationMessage(errorKey));
      }
    }
  }
}
```

### Internationalization (i18n)

```typescript
// en.json
{
  "errors": {
    // General validation
    "validation.failed": "Please check your input and try again.",
    "validation.invalid_page": "Invalid page number.",
    "validation.invalid_limit": "Invalid limit parameter.",
    
    // Field-specific validation (examples - add as needed)
    "validation.required.title": "Title is required.",
    "validation.required.email": "Email is required.",
    "validation.required.images": "At least one image is required.",
    "validation.min.title": "Title must be at least 3 characters.",
    "validation.max.description": "Description cannot exceed 500 characters.",
    "validation.url.images": "Please provide a valid image URL.",
    "validation.email.email": "Please provide a valid email address.",
    
    // Generic patterns (fallback for any field)
    "validation.required": "{{field}} is required.",
    "validation.min": "{{field}} must be at least {{param}} characters.",
    "validation.max": "{{field}} cannot exceed {{param}} characters.",
    "validation.url": "{{field}} must be a valid URL.",
    "validation.email": "{{field}} must be a valid email address.",
    
    // Auth & other errors
    "auth.unauthorized": "Please log in to continue.",
    "auth.forbidden": "You don't have permission to perform this action.",
    "gallery.not_found": "Gallery not found.",
    "server.internal_error": "An unexpected error occurred."
  }
}

// es.json (Spanish)
{
  "errors": {
    "validation.failed": "Por favor, verifica tu entrada e inténtalo de nuevo.",
    "validation.required.title": "El título es obligatorio.",
    "validation.required.email": "El correo electrónico es obligatorio.",
    "validation.min.title": "El título debe tener al menos 3 caracteres.",
    "validation.url.images": "Por favor proporciona una URL de imagen válida.",
    "validation.email.email": "Por favor proporciona un correo electrónico válido.",
    
    // Generic patterns
    "validation.required": "{{field}} es obligatorio.",
    "validation.min": "{{field}} debe tener al menos {{param}} caracteres.",
    "validation.url": "{{field}} debe ser una URL válida.",
    
    "auth.unauthorized": "Por favor, inicia sesión para continuar.",
    "gallery.not_found": "Galería no encontrada.",
    "server.internal_error": "Ocurrió un error inesperado."
  }
}

// Usage with i18next
import { useTranslation } from 'react-i18next';

function parseErrorKey(errorKey: string) {
  const match = errorKey.match(/^validation\.(\w+)\.(.+)$/);
  if (match) {
    const [, rule, field] = match;
    return { rule, field: field.replace(/_/g, ' ') };
  }
  return null;
}

function useErrorMessage(errorKey: string, param?: string) {
  const { t } = useTranslation();
  
  // Try specific key first
  if (t(`errors.${errorKey}`) !== `errors.${errorKey}`) {
    return t(`errors.${errorKey}`, { param });
  }
  
  // Try generic pattern for validation errors
  const parsed = parseErrorKey(errorKey);
  if (parsed) {
    const genericKey = `errors.validation.${parsed.rule}`;
    if (t(genericKey) !== genericKey) {
      return t(genericKey, { field: parsed.field, param });
    }
  }
  
  // Fallback
  return t('errors.validation.failed');
}

// Usage
const errorMessage = useErrorMessage('validation.required.title');
// Returns: "Title is required." (en) or "El título es obligatorio." (es)
```

### Error-Specific UI Actions

```typescript
function handleError(errorKey: string, status: number) {
  switch (errorKey) {
    case 'auth.unauthorized':
    case 'auth.token_expired':
      // Redirect to login
      router.push('/login');
      break;
      
    case 'auth.forbidden':
      // Show upgrade prompt
      showUpgradeModal();
      break;
      
    case 'gallery.not_found':
    case 'todo.not_found':
      // Redirect to list page
      router.push('/galleries');
      break;
      
    case 'server.rate_limit_exceeded':
      // Show retry with backoff
      showRetryDialog(30000); // Retry after 30 seconds
      break;
      
    case 'validation.failed':
      // Keep form open with error highlights
      // No redirect needed
      break;
      
    default:
      // Generic error handling
      if (status >= 500) {
        showErrorPage('Something went wrong on our end.');
      } else {
        showErrorToast('Please try again.');
      }
  }
}
```

### Error Analytics

```typescript
// Track error patterns
function trackError(errorKey: string, status: number, endpoint: string) {
  analytics.track('API Error', {
    error_key: errorKey,
    status_code: status,
    endpoint: endpoint,
    timestamp: new Date().toISOString(),
    user_id: getCurrentUserId(),
  });
  
  // Alert on critical errors
  if (status >= 500) {
    sentry.captureMessage(`API Error: ${errorKey}`, {
      level: 'error',
      extra: { status, endpoint },
    });
  }
}
```

## Best Practices

### For Frontend Developers

1. **Always check the error key**: Don't rely solely on HTTP status codes
2. **Provide fallback messages**: Always have a default error message
3. **Localize error messages**: Use i18n libraries for multi-language support
4. **Log error keys**: Include error keys in your logging for debugging
5. **Handle errors gracefully**: Provide helpful UI feedback based on error type
6. **Don't expose technical details**: Map technical keys to user-friendly messages

### For Backend Developers

1. **Always use error keys**: Every error should have an appropriate key
2. **Be specific**: Use domain-specific keys when available (e.g., `gallery.not_found` instead of `resource.not_found`)
3. **Be consistent**: Use the same key for the same error across all endpoints
4. **Document new keys**: Add new error keys to this documentation
5. **Test error scenarios**: Ensure error keys are correctly returned in all failure cases

## Example API Responses

### Successful Response
```http
HTTP/1.1 200 OK
Content-Type: application/json

{
  "data": {
    "id": "gallery-123",
    "title": "My Gallery",
    "images": [...]
  }
}
```

### Validation Error
```http
HTTP/1.1 422 Unprocessable Entity
Content-Type: application/json

{
  "error": "validation.failed"
}
```

### Not Found Error
```http
HTTP/1.1 404 Not Found
Content-Type: application/json

{
  "error": "gallery.not_found"
}
```

### Internal Server Error
```http
HTTP/1.1 500 Internal Server Error
Content-Type: application/json

{
  "error": "database.query_failed"
}
```

### Authentication Error
```http
HTTP/1.1 401 Unauthorized
Content-Type: application/json

{
  "error": "auth.unauthorized"
}
```

## Migration Guide

If you're migrating from an older error format, here's how to adapt:

### Old Format (with detailed messages)
```json
{
  "error": "failed to connect to database: connection timeout at host 10.0.1.5",
  "code": 500
}
```

### New Format (error key only)
```json
{
  "error": "database.connection_failed"
}
```

**Frontend changes:**
- Parse the error key instead of showing the raw message
- Map error keys to user-friendly messages in your UI layer
- Use the HTTP status code from the response headers for error categorization

## Additional Resources

- **Error Key Definitions**: `shared/errkey/errkey.go`
- **Failure Package**: `shared/failure/failure.go`
- **Response Handler**: `transport/http/response/response.go`
- **API Tests**: See handler tests for examples of all error keys in action

## Support

For questions about error keys or to request new error keys, please:
1. Check this documentation first
2. Review existing error keys in `shared/errkey/errkey.go`
3. Open an issue or PR if you need a new error key added
