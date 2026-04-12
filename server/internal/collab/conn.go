package collab

import (
	"context"

	"github.com/gorilla/websocket"
	"github.com/reearth/reearth/server/internal/usecase"
	"github.com/reearth/reearth/server/pkg/id"
	"golang.org/x/time/rate"
)

// Conn is one WebSocket client in a project room.
type Conn struct {
	hub       *Hub
	ws        *websocket.Conn
	projectID string
	sceneID   id.SceneID
	userID    string
	clientID  string
	photoURL  string
	operator  *usecase.Operator
	bgCtx     context.Context
	send      chan []byte
}

func (c *Conn) readPump(maxBytes int, lim *rate.Limiter, onMessage func([]byte) error) {
	defer func() {
		c.hub.unregister(c)
		_ = c.ws.Close()
	}()
	c.ws.SetReadLimit(int64(maxBytes))
	for {
		_, msg, err := c.ws.ReadMessage()
		if err != nil {
			return
		}
		if lim != nil && !lim.Allow() {
			return
		}
		if err := onMessage(msg); err != nil {
			return
		}
	}
}

func (c *Conn) writePump() {
	defer func() {
		_ = c.ws.Close()
	}()
	for msg := range c.send {
		if err := c.ws.WriteMessage(websocket.TextMessage, msg); err != nil {
			return
		}
	}
}
