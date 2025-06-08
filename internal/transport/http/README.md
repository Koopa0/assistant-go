# API Package

The api package provides common utilities and shared functionality for HTTP services in the Assistant application.

## Features

### Response Handling
- `ResponseWriter` - Standardized JSON response handling
- Error response formatting with timestamps
- Success response wrapping

### Request Parsing
- `ParseJSONRequest` - Parse JSON request bodies
- Query parameter helpers:
  - `QueryParamInt` - Parse integer query parameters with defaults
  - `QueryParamString` - Get string query parameters with defaults
  - `QueryParamBool` - Parse boolean query parameters with defaults

### Utilities
- `FilterEmptyStrings` - Remove empty strings from slices

## Usage

```go
import "github.com/koopa0/assistant-go/internal/api"

// In your HTTP handler
type HTTPHandler struct {
    service *YourService
    rw      *api.ResponseWriter
}

func NewHTTPHandler(service *YourService) *HTTPHandler {
    return &HTTPHandler{
        service: service,
        rw:      api.NewResponseWriter(),
    }
}

func (h *HTTPHandler) HandleRequest(w http.ResponseWriter, r *http.Request) {
    // Parse query parameters
    limit := api.QueryParamInt(r, "limit", 100)
    
    // Parse request body
    var req RequestType
    if err := api.ParseJSONRequest(r, &req); err != nil {
        h.rw.WriteError(w, "INVALID_REQUEST", err.Error(), http.StatusBadRequest)
        return
    }
    
    // Process request...
    result, err := h.service.Process(req)
    if err != nil {
        h.rw.WriteError(w, "SERVER_ERROR", "Failed to process", http.StatusInternalServerError)
        return
    }
    
    // Write success response
    h.rw.WriteSuccess(w, result)
}
```

## Migration Guide

When migrating existing handlers to use this package:

1. Add `rw *api.ResponseWriter` to your handler struct
2. Initialize it in your constructor with `api.NewResponseWriter()`
3. Replace `writeJSON` calls with `h.rw.WriteJSON`
4. Replace `writeError` calls with `h.rw.WriteError`
5. Replace manual query parameter parsing with the helper functions
6. Remove duplicate helper methods from your handler