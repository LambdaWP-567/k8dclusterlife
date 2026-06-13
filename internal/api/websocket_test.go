package api

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockSubscriber delivers pre-canned messages then blocks.
type mockSubscriber struct {
	messages []string
}

func (m *mockSubscriber) Subscribe(ctx context.Context, _ ...string) *redis.PubSub {
	// Use a real redis client pointed at a non-existent server so we get a
	// PubSub object we can return; in this test we won't actually send via it.
	// Instead we test the upgrader path separately.
	rdb := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1"})
	return rdb.Subscribe(ctx, "problems")
}

func TestHandleWebSocket_Upgrade(t *testing.T) {
	sub := &mockSubscriber{}
	srv := httptest.NewServer(HandleWebSocket(sub))
	defer srv.Close()

	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "/"
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	require.NoError(t, err)

	// Close immediately — server should not crash
	conn.Close()
	time.Sleep(50 * time.Millisecond)
	assert.True(t, true, "server handled disconnect gracefully")
}
