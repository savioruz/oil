# API Error Keys Documentation

## Overview

All API error responses return machine-readable error keys. This allows frontends to:
- Implement consistent error handling across all endpoints
- Provide translated error messages based on user locale
- Display appropriate UI for specific error types
- Log and track error patterns

## Response Format

Error responses use two different formats depending on the error type:

### Validation Errors (422 Unprocessable Entity)

Validation errors return an `errors` array containing all validation failures:

```json
{
  "errors": [
    {
      "field": "title",
      "message": "Title is required"
    },
    {
      "field": "email",
      "message": "Email must be a valid email address"
    }
  ]
}
```

Each error object contains:
- `field`: The name of the field that failed validation (in snake_case, e.g., "title", "email", "images[0]")
- `message`: A human-readable error message describing what went wrong

This format allows the API to return **all validation errors at once**, enabling frontends to highlight all problematic fields in a single response.

### Non-Validation Errors (4xx, 5xx)

All other errors return a single `error` field:

```json
{
  "error": "error.key.here"
}
```

The HTTP status code provides the error category (4xx for client errors, 5xx for server errors), and the error key provides specific details about what went wrong.

### Examples

**Validation Error (Single Field):**
```http
POST /api/galleries
422 Unprocessable Entity

{
  "errors": [
    {
      "field": "title",
      "message": "Title is required"
    }
  ]
}
```

**Validation Error (Multiple Fields):**
```http
POST /api/galleries
422 Unprocessable Entity

{
  "errors": [
    {
      "field": "title",
      "message": "Title is required"
    },
    {
      "field": "images",
      "message": "Images must be a valid URL"
    }
  ]
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
// Error handler for API calls
async function handleApiCall<T>(apiCall: Promise<T>): Promise<T> {
  try {
    return await apiCall;
  } catch (error: any) {
    // Handle validation errors (422)
    if (error.response?.data?.errors) {
      // Array of field errors
      const fieldErrors = error.response.data.errors;
      
      // Show first error or all errors
      const firstError = fieldErrors[0];
      toast.error(firstError.message);
      
      // Log all errors for debugging
      console.error('Validation errors:', fieldErrors);
      
      throw error;
    }
    
    // Handle other errors (error key format)
    if (error.response?.data?.error) {
      const errorKey = error.response.data.error;
      const message = translateErrorKey(errorKey);
      
      toast.error(message);
      console.error(`API Error [${errorKey}]:`, error);
      
      throw new Error(message);
    }
    
    throw error;
  }
}

// Translate error keys to user-friendly messages
function translateErrorKey(errorKey: string): string {
  const ERROR_MESSAGES: Record<string, string> = {
    'auth.unauthorized': 'Please log in to continue.',
    'auth.forbidden': 'You don\'t have permission to perform this action.',
    'gallery.not_found': 'Gallery not found.',
    'todo.not_found': 'Todo item not found.',
    'database.query_failed': 'A system error occurred. Please try again later.',
    'server.internal_error': 'An unexpected error occurred. Please try again later.',
  };
  
  return ERROR_MESSAGES[errorKey] || 'An error occurred. Please try again.';
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
// Handle form submission with field-specific errors
async function handleFormSubmit(data: GalleryFormData) {
  try {
    await createGallery(data);
    toast.success('Gallery created successfully!');
    router.push('/galleries');
  } catch (error: any) {
    // Handle validation errors
    if (error.response?.data?.errors) {
      // errors is an array: [{ field: "title", message: "Title is required" }]
      error.response.data.errors.forEach(fieldError => {
        // Set field-specific error in form (e.g., using react-hook-form)
        setError(fieldError.field, {
          type: 'server',
          message: fieldError.message, // Already human-readable!
        });
      });
    } else if (error.response?.data?.error) {
      // Handle non-validation errors
      toast.error(translateErrorKey(error.response.data.error));
    }
  }
}

// With React Hook Form
import { useForm } from 'react-hook-form';

function GalleryForm() {
  const { register, handleSubmit, setError, formState: { errors } } = useForm();

  const onSubmit = async (data) => {
    try {
      await createGallery(data);
      toast.success('Gallery created!');
    } catch (error: any) {
      if (error.response?.data?.errors) {
        // Highlight all invalid fields at once
        error.response.data.errors.forEach(({ field, message }) => {
          setError(field, { type: 'server', message });
        });
      }
    }
  };

  return (
    <form onSubmit={handleSubmit(onSubmit)}>
      <input {...register('title')} />
      {errors.title && <span>{errors.title.message}</span>}
      
      <input {...register('email')} />
      {errors.email && <span>{errors.email.message}</span>}
      
      <button type="submit">Create</button>
    </form>
  );
}
```

### Internationalization (i18n)

Since validation messages from the API are already in English, you have two options for internationalization:

**Option 1: Client-side translation (Recommended for flexibility)**

The API returns English messages, and your frontend translates them:

```typescript
// en.json
{
  "validation": {
    "{field} is required": "{{field}} is required",
    "{field} must be a valid email address": "{{field}} must be a valid email address",
    "{field} must be a valid URL": "{{field}} must be a valid URL",
    "{field} must be greater than or equal to {param}": "{{field}} must be at least {{param}}"
  },
  "errors": {
    "auth.unauthorized": "Please log in to continue",
    "gallery.not_found": "Gallery not found",
    "server.internal_error": "An unexpected error occurred"
  }
}

// es.json (Spanish)
{
  "validation": {
    "{field} is required": "{{field}} es obligatorio",
    "{field} must be a valid email address": "{{field}} debe ser un correo electrónico válido",
    "{field} must be a valid URL": "{{field}} debe ser una URL válida",
    "{field} must be greater than or equal to {param}": "{{field}} debe ser al menos {{param}}"
  },
  "errors": {
    "auth.unauthorized": "Por favor, inicia sesión para continuar",
    "gallery.not_found": "Galería no encontrada",
    "server.internal_error": "Ocurrió un error inesperado"
  }
}

// Usage with i18next
import { useTranslation } from 'react-i18next';

function useHandleApiError() {
  const { t } = useTranslation();
  
  return (error: any) => {
    // Validation errors
    if (error.response?.data?.errors) {
      return error.response.data.errors.map(({ field, message }) => ({
        field,
        message: t(`validation.${message}`, { 
          field: field.replace(/_/g, ' '),
          defaultValue: message // Fallback to English
        })
      }));
    }
    
    // Other errors  
    if (error.response?.data?.error) {
      const errorKey = error.response.data.error;
      return t(`errors.${errorKey}`, { defaultValue: 'An error occurred' });
    }
    
    return 'An unexpected error occurred';
  };
}

// Usage in component
function GalleryForm() {
  const handleError = useHandleApiError();
  
  const onSubmit = async (data) => {
    try {
      await createGallery(data);
    } catch (error) {
      const errors = handleError(error);
      // Display translated errors
      if (Array.isArray(errors)) {
        errors.forEach(({ field, message }) => {
          setError(field, { type: 'server', message });
        });
      } else {
        toast.error(errors);
      }
    }
  };
  
  return <form onSubmit={handleSubmit(onSubmit)}>...</form>;
}
```

**Option 2: Use messages as-is (Simpler, English only)**

If you only support English, use the API messages directly without translation:

```typescript
// Simply display the message from the API
try {
  await createGallery(data);
} catch (error: any) {
  if (error.response?.data?.errors) {
    error.response.data.errors.forEach(({ field, message }) => {
      setError(field, { type: 'server', message }); // message is already human-readable
    });
  }
}
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
