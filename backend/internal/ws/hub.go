package ws

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
)

type Message struct {
	Channel string      `json:"channel"`
	Target  string      `json:"target"`
	Title   string      `json:"title"`
	Content string      `json:"content"`
	Data    interface{} `json:"data,omitempty"`
}

type Client struct {
	Conn    *websocket.Conn
	User    string
	RoleSet map[string]struct{}
	DeptSet map[string]struct{}
	Send    chan Message
}

type Hub struct {
	mutex   sync.RWMutex
	clients map[*Client]struct{}
	redis   *redis.Client
	topic   string
}

func NewHub(redisClient *redis.Client, topic string) *Hub {
	h := &Hub{
		clients: make(map[*Client]struct{}),
		redis:   redisClient,
		topic:   topic,
	}
	if redisClient != nil {
		go h.consumeRedis()
	}
	return h
}

func (h *Hub) Register(c *Client) {
	h.mutex.Lock()
	h.clients[c] = struct{}{}
	h.mutex.Unlock()
}

func (h *Hub) Unregister(c *Client) {
	h.mutex.Lock()
	delete(h.clients, c)
	h.mutex.Unlock()
	close(c.Send)
	_ = c.Conn.Close()
}

func (h *Hub) Publish(msg Message) {
	h.broadcast(msg)
	if h.redis == nil {
		return
	}
	raw, err := json.Marshal(msg)
	if err != nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = h.redis.Publish(ctx, h.topic, raw).Err()
}

func (h *Hub) broadcast(msg Message) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	for client := range h.clients {
		if !allow(client, msg) {
			continue
		}
		select {
		case client.Send <- msg:
		default:
		}
	}
}

func allow(client *Client, msg Message) bool {
	switch msg.Channel {
	case "user":
		return client.User == msg.Target
	case "role":
		_, ok := client.RoleSet[msg.Target]
		return ok
	case "department":
		_, ok := client.DeptSet[msg.Target]
		return ok
	default:
		return true
	}
}

func (h *Hub) consumeRedis() {
	if h.redis == nil {
		return
	}
	sub := h.redis.Subscribe(context.Background(), h.topic)
	defer func() { _ = sub.Close() }()
	ch := sub.Channel()
	for msg := range ch {
		var m Message
		if err := json.Unmarshal([]byte(msg.Payload), &m); err != nil {
			log.Printf("ws redis unmarshal error: %v", err)
			continue
		}
		h.broadcast(m)
	}
}
