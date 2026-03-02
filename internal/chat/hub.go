package chat

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

const (
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10
	maxMsgSize = 32 << 10
)

// Client is a single WebSocket connection bound to a user.
type Client struct {
	UserID uuid.UUID
	Conn   *websocket.Conn
	Send   chan []byte
	Hub    *Hub
	logger *zap.Logger
}

// ReadPump reads messages from the connection (blocking). Call in a goroutine or after WritePump.
func (c *Client) ReadPump() {
	defer func() {
		c.Hub.Unregister(c)
		_ = c.Conn.Close()
	}()
	c.Conn.SetReadLimit(maxMsgSize)
	_ = c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error {
		_ = c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	for {
		_, raw, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.logger.Debug("chat ws read error", zap.Error(err))
			}
			break
		}
		var envelope struct {
			Type string          `json:"type"`
			Data json.RawMessage `json:"data"`
		}
		if err := json.Unmarshal(raw, &envelope); err != nil {
			continue
		}
		switch envelope.Type {
		case "typing":
			var body struct {
				ConversationID string `json:"conversation_id"`
			}
			if json.Unmarshal(envelope.Data, &body) == nil && body.ConversationID != "" {
				convID, _ := uuid.Parse(body.ConversationID)
				c.Hub.BroadcastTyping(convID, c.UserID)
			}
		case "message":
			// Persistence via REST; WS only receives broadcasts.
		default:
			// ignore
		}
	}
}

// WritePump writes messages to the connection. Run in a goroutine.
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		_ = c.Conn.Close()
	}()
	for {
		select {
		case msg, ok := <-c.Send:
			_ = c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.Conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// OnTypingPeer resolves peer user ID for a conversation; used to send typing to the right user.
type OnTypingPeer func(conversationID, fromUserID uuid.UUID) (peerID uuid.UUID, ok bool)

// Hub holds all connected clients by user ID and broadcasts to conversations.
type Hub struct {
	mu       sync.RWMutex
	clients  map[uuid.UUID][]*Client
	presence *PresenceStore
	onTyping OnTypingPeer
	logger   *zap.Logger
}

func NewHub(presence *PresenceStore, logger *zap.Logger) *Hub {
	return &Hub{
		clients:  make(map[uuid.UUID][]*Client),
		presence: presence,
		logger:   logger,
	}
}

// SetOnTyping sets callback to resolve conversation peer for typing events.
func (h *Hub) SetOnTyping(f OnTypingPeer) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.onTyping = f
}

func (h *Hub) Register(userID uuid.UUID, conn *websocket.Conn) *Client {
	c := &Client{
		UserID: userID,
		Conn:   conn,
		Send:   make(chan []byte, 256),
		Hub:    h,
		logger: h.logger,
	}
	h.mu.Lock()
	h.clients[userID] = append(h.clients[userID], c)
	h.mu.Unlock()
	if h.presence != nil {
		_ = h.presence.SetOnline(context.Background(), userID)
	}
	return c
}

// Unregister removes client and sets offline if no more connections for user.
func (h *Hub) Unregister(c *Client) {
	h.mu.Lock()
	list := h.clients[c.UserID]
	for i, cl := range list {
		if cl == c {
			list = append(list[:i], list[i+1:]...)
			break
		}
	}
	if len(list) == 0 {
		delete(h.clients, c.UserID)
		if h.presence != nil {
			_ = h.presence.SetOffline(context.Background(), c.UserID)
		}
	} else {
		h.clients[c.UserID] = list
	}
	h.mu.Unlock()
	close(c.Send)
}

// SendToUser sends payload to all connections of the user.
func (h *Hub) SendToUser(userID uuid.UUID, payload []byte) {
	h.mu.RLock()
	list := h.clients[userID]
	h.mu.RUnlock()
	for _, c := range list {
		select {
		case c.Send <- payload:
		default:
			// skip if buffer full
		}
	}
}

// BroadcastToConversation sends payload to both participants (userA, userB).
func (h *Hub) BroadcastToConversation(userAID, userBID uuid.UUID, payload []byte) {
	h.SendToUser(userAID, payload)
	h.SendToUser(userBID, payload)
}

// BroadcastTyping notifies the other participant in the conversation (uses OnTypingPeer to resolve peer).
func (h *Hub) BroadcastTyping(conversationID, fromUserID uuid.UUID) {
	h.mu.RLock()
	f := h.onTyping
	h.mu.RUnlock()
	if f == nil {
		return
	}
	peerID, ok := f(conversationID, fromUserID)
	if !ok {
		return
	}
	out := map[string]interface{}{
		"type": "typing",
		"data": map[string]string{
			"conversation_id": conversationID.String(),
			"user_id":         fromUserID.String(),
		},
	}
	raw, _ := json.Marshal(out)
	h.SendToUser(peerID, raw)
}

// BroadcastMessage sends a message event to both participants. Conversation is (userAID, userBID).
func (h *Hub) BroadcastMessage(userAID, userBID uuid.UUID, msg *Message) {
	payload, _ := json.Marshal(map[string]interface{}{
		"type": "message",
		"data": msg,
	})
	h.BroadcastToConversation(userAID, userBID, payload)
}
