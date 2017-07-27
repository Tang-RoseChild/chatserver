package message

import (
	"time"
)

const (
	MsgTypeChat = 1
	MsgTypeInfo = 4
)

type Message struct {
	Type    int      `json:"type"`
	Payload *Payload `json:"payload"`
}

type Payload struct {
	ChatMessage *ChatMessage `json:"chatMessage,omitempty"`
	InfoMessage *InfoMessage `json:"infoMessage,omitempty"`
}

type ChatMessage struct {
	From      string    `json:"from"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

type InfoMessage struct {
	OnlineCount int `json:"onlineCount"`
}
