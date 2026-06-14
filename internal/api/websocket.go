package api

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
)

// EventSubscriber can subscribe to cluster events via Redis Pub/Sub.
type EventSubscriber interface {
	Subscribe(ctx context.Context, channels ...string) *redis.PubSub
}

var wsUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// HandleWebSocket upgrades HTTP to WebSocket and forwards Redis pub/sub events.
func HandleWebSocket(sub EventSubscriber) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := wsUpgrader.Upgrade(w, r, nil)
		if err != nil {
			slog.Warn("ws upgrade failed", "error", err)
			return
		}
		defer conn.Close()

		ctx, cancel := context.WithCancel(r.Context())
		defer cancel()

		// Pump Redis messages → WebSocket
		ps := sub.Subscribe(ctx, "problems")
		defer ps.Close()

		// Close context when client disconnects
		go func() {
			for {
				if _, _, err := conn.ReadMessage(); err != nil {
					cancel()
					return
				}
			}
		}()

		ch := ps.Channel()
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-ch:
				if !ok {
					return
				}
				if err := conn.WriteMessage(websocket.TextMessage, []byte(msg.Payload)); err != nil {
					return
				}
			}
		}
	}
}
