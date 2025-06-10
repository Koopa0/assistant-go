// Package transport provides implementations for various network transport
// protocols used by the assistant to communicate with clients.
//
// Sub-packages like 'http', 'sse' (Server-Sent Events), and 'websocket'
// contain specific handlers and logic for establishing and managing
// connections, and for message passing over these respective protocols.
// This package abstracts the raw network communication details from the
// core application logic.
package transport
