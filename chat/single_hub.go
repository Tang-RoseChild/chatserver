package chat

import (
	"time"

	"github.com/Tang-RoseChild/chatserver/message"
)

type SingleHub Hub

func (h *SingleHub) Run() {
	go func() {
		for {
			select {
			case <-h.done:
				return
			case msg := <-h.pub:
				h.broadcast <- msg
			}
		}
	}()
	go h.broadcastOnlineCount()
	h.messageHandler()
}

func (h *SingleHub) broadcastOnlineCount() {
	ticker := time.NewTicker(time.Second)
	for {
		select {
		case <-h.done:
			return
		case <-ticker.C:
			h.broadcast <- &message.Message{
				Type:    message.MsgTypeInfo,
				Payload: &message.Payload{InfoMessage: &message.InfoMessage{OnlineCount: len(h.clients)}}}
		}
	}
}

func (h *SingleHub) messageHandler() {
	(*Hub)(h).messageHandler()
}
