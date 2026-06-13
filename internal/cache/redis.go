package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const problemsTTL = 5 * time.Minute

type Client struct {
	rdb *redis.Client
}

func New(addr string) *Client {
	rdb := redis.NewClient(&redis.Options{
		Addr:         addr,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     10,
	})
	return &Client{rdb: rdb}
}

func (c *Client) Ping(ctx context.Context) error {
	return c.rdb.Ping(ctx).Err()
}

func (c *Client) Close() error {
	return c.rdb.Close()
}

func (c *Client) SetProblems(ctx context.Context, clusterID string, problems any) error {
	data, err := json.Marshal(problems)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	key := "problems:" + clusterID
	return c.rdb.Set(ctx, key, data, problemsTTL).Err()
}

func (c *Client) GetProblems(ctx context.Context, clusterID string, out any) error {
	key := "problems:" + clusterID
	data, err := c.rdb.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil
	}
	if err != nil {
		return err
	}
	return json.Unmarshal(data, out)
}

func (c *Client) DeleteProblems(ctx context.Context, clusterID string) error {
	return c.rdb.Del(ctx, "problems:"+clusterID).Err()
}

// Publish sends an event to all WebSocket subscribers.
func (c *Client) Publish(ctx context.Context, channel string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return c.rdb.Publish(ctx, channel, data).Err()
}

// Subscribe returns a subscription for live events.
func (c *Client) Subscribe(ctx context.Context, channels ...string) *redis.PubSub {
	return c.rdb.Subscribe(ctx, channels...)
}
