package websocket

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/soumya-codes/realtime-hub/hubserver/internal/message"
	"go.uber.org/zap"
)

const (
	writeWait      = 1 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512
)

// Connection represents the WebSocket connection.
type Connection struct {
	id string
	ws *websocket.Conn

	// Buffered read and write channel to hold messages
	readCh  chan []byte
	writeCh chan message.MessageDetails

	logger *zap.Logger
	closed bool
	mu     sync.Mutex
}

// Upgrader to upgrade HTTP connections to WebSocket connections
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Upgrade upgrades an HTTP connection to a WebSocket connection and assigns a unique id to the connection.
func Upgrade(w http.ResponseWriter, r *http.Request, h *MessageHandler) (*Connection, error) {
	logger := h.logger
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Error("Failed to upgrade to WebSocket connection", zap.Error(err))
		return nil, fmt.Errorf("failed to upgrade to WebSocket connection: %w", err)
	}

	conn := &Connection{
		id: uuid.New().String(),
		ws: ws,

		readCh:  make(chan []byte, 256),
		writeCh: make(chan message.MessageDetails, 256),
		logger:  logger,
	}

	go conn.readPump(h)
	go conn.writePump(h)

	return conn, nil
}

// readPump handles reading messages from the WebSocket connection
func (c *Connection) readPump(h *MessageHandler) {
	defer func() {
		h.remove <- c.id
	}()

	c.ws.SetReadLimit(maxMessageSize)
	err := c.ws.SetReadDeadline(time.Now().Add(pongWait))
	if err != nil {
		c.logger.Error("Error setting read deadline", zap.String("conn-id", c.id), zap.Error(err))
		return
	}

	c.ws.SetPongHandler(func(string) error {
		err := c.ws.SetReadDeadline(time.Now().Add(pongWait))
		if err != nil {
			c.logger.Error("Error extending read deadline", zap.String("conn-id", c.id), zap.Error(err))
			return err
		}
		return nil
	})

	for {
		_, message, err := c.ws.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.logger.Error("Unexpected close error", zap.String("conn-id", c.id), zap.Error(err))
			} else {
				c.logger.Error("Error reading message", zap.String("conn-id", c.id), zap.Error(err))
			}
			return
		}
		c.readCh <- message
	}
}

// writePump handles writing messages to the WebSocket connection
func (c *Connection) writePump(h *MessageHandler) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		h.remove <- c.id
	}()

	for {
		select {
		case md, ok := <-c.writeCh:
			if !ok {
				err := c.ws.WriteMessage(websocket.CloseMessage, []byte{})
				if err != nil {
					c.logger.Error("Error closing connection", zap.String("conn-id", c.id), zap.Error(err))
				}
				return
			}

			if err := c.ws.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				c.logger.Error("Error setting write deadline", zap.String("conn-id", c.id), zap.Error(err))
				return
			}

			if err := c.ws.WriteMessage(websocket.TextMessage, md.Message); err != nil {
				c.logger.Error("Error sending message to the client", zap.String("conn-id", c.id), zap.Error(err))
				return
			}

		case <-ticker.C:
			if err := c.ws.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				c.logger.Error("Error setting write deadline for ping message", zap.String("conn-id", c.id), zap.Error(err))
				return
			}

			if err := c.ws.WriteMessage(websocket.PingMessage, nil); err != nil {
				c.logger.Error("Error pinging the client", zap.String("conn-id", c.id), zap.Error(err))
				return
			}
		}
	}
}

// Close closes the WebSocket connection and the related channels.
func (c *Connection) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}

	close(c.writeCh)
	close(c.readCh)
	err := c.ws.Close()
	if err != nil {
		c.logger.Error("Error closing connection", zap.String("conn-id", c.id), zap.Error(err))
		return fmt.Errorf("error closing connection: %w", err)
	}

	c.closed = true
	return nil
}
