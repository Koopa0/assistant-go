// Package ai provides services for interacting with underlying AI models
// from various providers (e.g., Claude, Gemini). It offers a unified
// interface for common AI operations like generating responses (blocking and
// streaming), generating embeddings, and managing prompts.
//
// The core 'Service' in this package orchestrates calls to specific provider
// clients (located in sub-packages like 'claude' and 'gemini'), handling
// request/response transformations and centralizing AI-related configurations.
// Sub-package 'prompt' offers capabilities for dynamic prompt construction
// and enhancement.
package ai
