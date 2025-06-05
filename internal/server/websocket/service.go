package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/koopa0/assistant-go/internal/assistant"
	"github.com/koopa0/assistant-go/internal/observability"
)

// WebSocketService 提供 WebSocket 連接管理
type WebSocketService struct {
	assistant *assistant.Assistant
	logger    *slog.Logger
	metrics   *observability.Metrics

	// 連接管理
	clients    map[string]*Client
	register   chan *Client
	unregister chan *Client
	broadcast  chan []byte
	mutex      sync.RWMutex

	// WebSocket 升級器
	upgrader websocket.Upgrader
}

// Client 代表一個 WebSocket 客戶端
type Client struct {
	ID       string
	UserID   string
	Conn     *websocket.Conn
	Send     chan []byte
	Service  *WebSocketService
	LastSeen time.Time
}

// Message WebSocket 訊息格式
type Message struct {
	Type      string                 `json:"type"`
	ID        string                 `json:"id,omitempty"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	UserID    string                 `json:"user_id,omitempty"`
}

// NewWebSocketService 建立新的 WebSocket 服務
func NewWebSocketService(assistant *assistant.Assistant, logger *slog.Logger, metrics *observability.Metrics) *WebSocketService {
	return &WebSocketService{
		assistant:  assistant,
		logger:     logger,
		metrics:    metrics,
		clients:    make(map[string]*Client),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan []byte),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				// TODO: 實現適當的來源檢查
				return true
			},
		},
	}
}

// Start 啟動 WebSocket 服務
func (s *WebSocketService) Start(ctx context.Context) {
	s.logger.Info("Starting WebSocket service")

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("WebSocket service stopping")
			return

		case client := <-s.register:
			s.registerClient(client)

		case client := <-s.unregister:
			s.unregisterClient(client)

		case message := <-s.broadcast:
			s.broadcastMessage(message)
		}
	}
}

// HandleWebSocket 處理 WebSocket 連接
func (s *WebSocketService) HandleWebSocket(w http.ResponseWriter, r *http.Request, userID string) error {
	// 升級到 WebSocket 連接
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Error("Failed to upgrade to WebSocket", slog.Any("error", err))
		return err
	}

	// 建立客戶端
	client := &Client{
		ID:       generateClientID(),
		UserID:   userID,
		Conn:     conn,
		Send:     make(chan []byte, 256),
		Service:  s,
		LastSeen: time.Now(),
	}

	// 註冊客戶端
	s.register <- client

	// 啟動讀寫協程
	go client.readPump()
	go client.writePump()

	s.logger.Debug("WebSocket connection established",
		slog.String("client_id", client.ID),
		slog.String("user_id", userID))

	return nil
}

// SendToUser 發送訊息給特定使用者
func (s *WebSocketService) SendToUser(userID string, message Message) error {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	messageBytes, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	sent := 0
	for _, client := range s.clients {
		if client.UserID == userID {
			select {
			case client.Send <- messageBytes:
				sent++
			default:
				s.logger.Warn("Client send channel full",
					slog.String("client_id", client.ID))
			}
		}
	}

	s.logger.Debug("Message sent to user",
		slog.String("user_id", userID),
		slog.Int("clients", sent))

	return nil
}

// Broadcast 廣播訊息給所有客戶端
func (s *WebSocketService) Broadcast(message Message) error {
	messageBytes, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	s.broadcast <- messageBytes
	return nil
}

// GetConnectedUsers 取得已連接的使用者
func (s *WebSocketService) GetConnectedUsers() []string {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	userSet := make(map[string]bool)
	for _, client := range s.clients {
		userSet[client.UserID] = true
	}

	users := make([]string, 0, len(userSet))
	for userID := range userSet {
		users = append(users, userID)
	}

	return users
}

// GetConnectionStats 取得連接統計
func (s *WebSocketService) GetConnectionStats() map[string]interface{} {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	userConnections := make(map[string]int)
	for _, client := range s.clients {
		userConnections[client.UserID]++
	}

	return map[string]interface{}{
		"total_connections":    len(s.clients),
		"unique_users":         len(userConnections),
		"connections_per_user": userConnections,
	}
}

// 內部方法

// registerClient 註冊客戶端
func (s *WebSocketService) registerClient(client *Client) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.clients[client.ID] = client

	s.logger.Debug("Client registered",
		slog.String("client_id", client.ID),
		slog.String("user_id", client.UserID),
		slog.Int("total_clients", len(s.clients)))

	// 發送歡迎訊息
	welcomeMsg := Message{
		Type: "welcome",
		Data: map[string]interface{}{
			"client_id":   client.ID,
			"server_time": time.Now().UTC().Format(time.RFC3339),
		},
		Timestamp: time.Now(),
	}

	if msgBytes, err := json.Marshal(welcomeMsg); err == nil {
		select {
		case client.Send <- msgBytes:
		default:
		}
	}
}

// unregisterClient 取消註冊客戶端
func (s *WebSocketService) unregisterClient(client *Client) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if _, exists := s.clients[client.ID]; exists {
		delete(s.clients, client.ID)
		close(client.Send)

		s.logger.Debug("Client unregistered",
			slog.String("client_id", client.ID),
			slog.String("user_id", client.UserID),
			slog.Int("total_clients", len(s.clients)))
	}
}

// broadcastMessage 廣播訊息
func (s *WebSocketService) broadcastMessage(message []byte) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	for clientID, client := range s.clients {
		select {
		case client.Send <- message:
		default:
			s.logger.Warn("Failed to send broadcast message",
				slog.String("client_id", clientID))
		}
	}
}

// Client 方法

// readPump 讀取客戶端訊息
func (c *Client) readPump() {
	defer func() {
		c.Service.unregister <- c
		c.Conn.Close()
	}()

	// 設置讀取超時
	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		c.LastSeen = time.Now()
		return nil
	})

	for {
		_, messageBytes, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.Service.logger.Error("WebSocket read error", slog.Any("error", err))
			}
			break
		}

		c.LastSeen = time.Now()
		c.handleMessage(messageBytes)
	}
}

// writePump 寫入訊息到客戶端
func (c *Client) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				c.Service.logger.Error("WebSocket write error", slog.Any("error", err))
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleMessage 處理客戶端訊息
func (c *Client) handleMessage(messageBytes []byte) {
	var msg Message
	if err := json.Unmarshal(messageBytes, &msg); err != nil {
		c.Service.logger.Warn("Invalid message format", slog.Any("error", err))
		return
	}

	c.Service.logger.Debug("Received message",
		slog.String("client_id", c.ID),
		slog.String("type", msg.Type))

	// 處理不同類型的訊息
	switch msg.Type {
	case "ping":
		c.sendPong()
	case "chat":
		c.handleChatMessage(msg)
	case "tool_execution":
		c.handleToolExecution(msg)
	default:
		c.Service.logger.Warn("Unknown message type", slog.String("type", msg.Type))
	}
}

// sendPong 發送 pong 回應
func (c *Client) sendPong() {
	pongMsg := Message{
		Type:      "pong",
		Timestamp: time.Now(),
	}

	if msgBytes, err := json.Marshal(pongMsg); err == nil {
		select {
		case c.Send <- msgBytes:
		default:
		}
	}
}

// handleChatMessage 處理聊天訊息
func (c *Client) handleChatMessage(msg Message) {
	// TODO: 整合 Assistant 聊天功能
	c.Service.logger.Debug("Handling chat message",
		slog.String("client_id", c.ID))
}

// handleToolExecution 處理工具執行
func (c *Client) handleToolExecution(msg Message) {
	// TODO: 整合工具執行功能
	c.Service.logger.Debug("Handling tool execution",
		slog.String("client_id", c.ID))
}

// 輔助函數

// generateClientID 生成客戶端 ID
func generateClientID() string {
	return fmt.Sprintf("client_%d", time.Now().UnixNano())
}
