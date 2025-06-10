// Package platform provides foundational services and infrastructure components
// for the application. These components are generally application-agnostic
// and could, in principle, support other business domains.
//
// Sub-packages include:
// - 'event': For application-wide event handling.
// - 'observability': For logging, metrics, and tracing.
// - 'ratelimit': For request rate limiting.
// - 'server': For HTTP server setup, routing, and middleware.
// - 'storage': For database interactions, primarily PostgreSQL.
package platform
