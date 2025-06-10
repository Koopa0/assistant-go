// Package assistant contains the core logic for the AI-powered development
// assistant. It is responsible for orchestrating user query processing,
// managing conversation context, integrating with AI services, and
// coordinating tool execution.
//
// The main 'Assistant' struct acts as a high-level facade, while the 'Processor'
// component within this package implements the detailed step-by-step pipeline
// for handling requests. This includes context enrichment, interaction with
// AI models (via the 'ai' package), tool invocation (via the 'tool' package),
// and conversation state management (via the 'conversation' package).
package assistant
