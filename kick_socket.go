package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gorilla/websocket"
)

const (
	wsURL       = "wss://ws-us2.pusher.com/app/32cbd69e4b950bf97679"
	subsFile    = "messages.json"
	maxBackoff  = 32 * time.Second
	initialBack = 1 * time.Second
)

var subs = []map[string]interface{}{ // subscription messages
	{"event": "pusher:subscribe", "data": map[string]string{"auth": "", "channel": "chatroom_13808"}},
	{"event": "pusher:subscribe", "data": map[string]string{"auth": "", "channel": "chatrooms.13808.v2"}},
	{"event": "pusher:subscribe", "data": map[string]string{"auth": "", "channel": "channel_13809"}},
	{"event": "pusher:subscribe", "data": map[string]string{"auth": "", "channel": "channel.13809"}},
	{"event": "pusher:subscribe", "data": map[string]string{"auth": "", "channel": "chatrooms.13808"}},
}

type rawMsg struct {
	Event   string `json:"event"`
	Data    string `json:"data"`
	Channel string `json:"channel"`
}

type ChatMessage struct {
	ID         string    `json:"id"`
	ChatroomID int       `json:"chatroom_id"`
	Content    string    `json:"content"`
	Type       string    `json:"type"`
	CreatedAt  time.Time `json:"created_at"`
	Sender     struct {
		ID       int    `json:"id"`
		Username string `json:"username"`
		Slug     string `json:"slug"`
		Identity struct {
			Color  string        `json:"color"`
			Badges []interface{} `json:"badges"`
		} `json:"identity"`
	} `json:"sender"`
	Metadata struct {
		MessageRef string `json:"message_ref"`
	} `json:"metadata"`
}

// connect opens a WebSocket and subscribes.
func connect() (*websocket.Conn, error) {
	c, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		return nil, err
	}
	for _, msg := range subs {
		if err := c.WriteJSON(msg); err != nil {
			c.Close()
			return nil, fmt.Errorf("subscribe error: %w", err)
		}
	}
	return c, nil
}

func main() {
	// prepare file for JSON lines
	f, err := os.OpenFile(subsFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal("file open:", err)
	}
	defer f.Close()
	encoder := json.NewEncoder(f)

	backoff := initialBack
	for {
		conn, err := connect()
		if err != nil {
			log.Printf("connect error: %v, retrying in %v", err, backoff)
			time.Sleep(backoff)
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
			continue
		}
		log.Println("connected and subscribed")
		backoff = initialBack // reset backoff on success

		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				log.Println("read error, reconnecting:", err)
				conn.Close()
				break
			}

			// parse outer
			var rm rawMsg
			if err := json.Unmarshal(msg, &rm); err != nil {
				log.Println("invalid raw message, skipping:", err)
				continue
			}

			if rm.Event != "App\\Events\\ChatMessageEvent" {
				continue
			}

			// parse inner data
			var chat ChatMessage
			if err := json.Unmarshal([]byte(rm.Data), &chat); err != nil {
				log.Println("invalid chat payload, skipping:", err)
				continue
			}

			// log and write
			log.Printf("[%s] %s: %s", rm.Channel, chat.Sender.Username, chat.Content)
			// write raw JSON line
			var raw interface{}
			if err := json.Unmarshal(msg, &raw); err == nil {
				encoder.Encode(raw)
			}
		}
	}
}
