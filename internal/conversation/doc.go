// Package conversation manages the lifecycle of conversations between users
// and the AI assistant. This includes creating conversations, adding messages,
// retrieving message history, and managing conversation metadata.
//
// It defines interfaces for conversation operations and provides services
// that interact with the database (via sqlc) to persist and retrieve
// conversation data. It aims to keep conversation state consistent and
// accessible to other parts of the system, like the assistant processor.
package conversation
