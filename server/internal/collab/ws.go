package collab

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/reearth/reearth/server/internal/adapter"
	"github.com/reearth/reearth/server/internal/app/config"
	"github.com/reearth/reearth/server/pkg/id"
	"golang.org/x/time/rate"
)

const (
	defaultMaxMessageBytes = 64 * 1024
	defaultMsgsPerSec      = 40
	sendChannelBuf         = 32
)

// ClientMessage is the minimal supported inbound protocol (v1).
type ClientMessage struct {
	V int             `json:"v"`
	T string          `json:"t"`
	D json.RawMessage `json:"d,omitempty"`
}

// ServeWS returns an Echo handler for GET /api/collab/ws?projectId=...
func ServeWS(hub *Hub, cfg *config.CollabConfig, allowedOrigins []string) echo.HandlerFunc {
	maxBytes := cfg.MaxMessageBytes
	if maxBytes <= 0 {
		maxBytes = defaultMaxMessageBytes
	}
	msgsPerSec := cfg.MaxMessagesPerSec
	if msgsPerSec <= 0 {
		msgsPerSec = defaultMsgsPerSec
	}

	up := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			if origin == "" {
				return true
			}
			for _, o := range allowedOrigins {
				if o != "" && o == origin {
					return true
				}
			}
			return false
		},
	}

	return func(c echo.Context) error {
		op := adapter.Operator(c.Request().Context())
		if op == nil {
			return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
		}

		pidStr := strings.TrimSpace(c.QueryParam("projectId"))
		if pidStr == "" {
			return echo.NewHTTPError(http.StatusBadRequest, "projectId is required")
		}
		pid, err := id.ProjectIDFrom(pidStr)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid projectId")
		}

		uc := adapter.Usecases(c.Request().Context())
		if uc == nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "internal error")
		}

		pj, err := uc.Project.FindActiveById(c.Request().Context(), pid, op)
		if err != nil || pj == nil {
			return echo.NewHTTPError(http.StatusForbidden, "project not accessible")
		}
		if !op.IsReadableScene(pj.Scene()) {
			return echo.NewHTTPError(http.StatusForbidden, "scene not readable")
		}

		ws, err := up.Upgrade(c.Response(), c.Request(), nil)
		if err != nil {
			return err
		}

		bgCtx := context.WithoutCancel(c.Request().Context())
		userID := ""
		photoURL := ""
		if u := adapter.User(bgCtx); u != nil {
			userID = u.ID().String()
			if md := u.Metadata(); md != nil {
				photoURL = md.PhotoURL()
			}
		}
		conn := &Conn{
			hub:       hub,
			ws:        ws,
			projectID: pid.String(),
			sceneID:   pj.Scene(),
			userID:    userID,
			photoURL:  photoURL,
			operator:  op,
			bgCtx:     bgCtx,
			send:      make(chan []byte, sendChannelBuf),
		}
		hub.register(conn)

		lim := rate.NewLimiter(rate.Limit(msgsPerSec), msgsPerSec)

		go conn.writePump()
		conn.readPump(maxBytes, lim, func(raw []byte) error {
			return handleClientMessage(hub, conn, raw, maxBytes)
		})
		return nil
	}
}

func handleClientMessage(hub *Hub, from *Conn, raw []byte, maxBytes int) error {
	ctx := from.bgCtx
	var m ClientMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		return err
	}
	if m.V != 1 {
		return fmt.Errorf("unsupported protocol version")
	}
	switch m.T {
	case "ping":
		resp, _ := json.Marshal(ClientMessage{V: 1, T: "pong"})
		select {
		case from.send <- resp:
		default:
		}
		return nil
	case "relay":
		if len(raw) > maxBytes {
			return errors.New("message too large")
		}
		hub.broadcastFromClient(ctx, from.projectID, raw, from)
		return nil
	case "apply":
		if len(raw) > maxBytes {
			return errors.New("message too large")
		}
		if len(m.D) == 0 {
			return fmt.Errorf("apply requires d")
		}
		return dispatchApply(ctx, hub, from, m.D)
	case "lock":
		if len(raw) > maxBytes {
			return errors.New("message too large")
		}
		if len(m.D) == 0 {
			return fmt.Errorf("lock requires d")
		}
		return dispatchLock(ctx, hub, from, m.D)
	case "chat":
		if len(raw) > maxBytes {
			return errors.New("message too large")
		}
		if len(m.D) == 0 {
			return fmt.Errorf("chat requires d")
		}
		return dispatchChat(ctx, hub, from, m.D)
	case "cursor":
		if len(raw) > maxBytes {
			return errors.New("message too large")
		}
		if len(m.D) == 0 {
			return fmt.Errorf("cursor requires d")
		}
		return dispatchCursor(ctx, hub, from, m.D)
	case "activity":
		if len(raw) > maxBytes {
			return errors.New("message too large")
		}
		if len(m.D) == 0 {
			return fmt.Errorf("activity requires d")
		}
		return dispatchActivity(ctx, hub, from, m.D)
	default:
		return fmt.Errorf("unknown message type")
	}
}
