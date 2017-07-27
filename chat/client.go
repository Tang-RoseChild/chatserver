package chat

import (
	"log"
	"sync"
	"time"

	"github.com/Tang-RoseChild/chatserver/message"
	"github.com/gorilla/websocket"
)

type client struct {
	conn         *websocket.Conn
	msg          chan *message.Message
	Hub          *Hub
	rwdeadline   int64 // seconds
	minPing      int64 // seconds
	maxPing      int64 // seconds
	lastActivity int64
	mux          sync.Mutex
	errChan      chan error
}

const (
	defaultRWDeadline     = 600
	defaultMinPing        = 10
	defaultMaxPing        = 30
	defaultMsgChanSize    = 126
	defaultMaxMsgPerFlush = 30
)

func NewClient(hub *Hub, conn *websocket.Conn) *client {
	return &client{
		conn:         conn,
		Hub:          hub,
		msg:          make(chan *message.Message, defaultMsgChanSize),
		rwdeadline:   defaultRWDeadline,
		minPing:      defaultMinPing,
		maxPing:      defaultMaxPing,
		lastActivity: time.Now().Unix(),
		errChan:      make(chan error),
	}
}

func (c *client) Serve() {
	ticker := time.NewTicker(time.Duration(c.minPing) * time.Second)
	defer func() {
		c.conn.Close()
		ticker.Stop()
		c.Hub.Leave(c)
	}()

	c.conn.SetPongHandler(func(str string) error {
		c.UpdateLastActivity(time.Now().Unix())
		return nil
	})

	go func() {
		for {
			msg, err := c.read()
			if err != nil {
				c.errChan <- err
				return
			}
			c.Hub.Pub(msg)
		}
	}()
	msgs := make([]*message.Message, defaultMaxMsgPerFlush)
	writeMsgTicker := time.NewTicker(20 * time.Millisecond)
	// writeMsgTicker := time.NewTicker(2 * time.Second)
	for {
		select {
		// case msg, ok := <-c.msg:
		case <-writeMsgTicker.C:
			n := len(c.msg)
			if n == 0 {
				continue
			}
			if n > defaultMaxMsgPerFlush {
				n = defaultMaxMsgPerFlush
			}
			for i := 0; i < n; i++ {
				msgs[i] = <-c.msg
			}

			if err := c.write(websocket.TextMessage, msgs[:n]); err != nil {
				log.Print("err ", err)
				return
			}
			// need to moidify activity
			c.UpdateLastActivity(time.Now().Unix())
		case <-ticker.C:
			status := c.ActiveStatus()
			switch status {
			case active:
			// do nothing
			case disconnected:
				return
			default:
				if err := c.write(websocket.PingMessage, nil); err != nil {
					log.Print("err ", err)
					return
				}
			}
		case err := <-c.errChan:
			// handle error
			log.Print("err ", err)
			return
		}
	}

}

const (
	active       = 1
	notActive    = 2
	disconnected = 3
)

func (c *client) UpdateLastActivity(now int64) {
	c.mux.Lock()
	if c.lastActivity < now {
		c.lastActivity = now
	}
	c.mux.Unlock()
}

func (c *client) ActiveStatus() int {
	now := time.Now().Unix()
	var status int
	c.mux.Lock()
	defer c.mux.Unlock()
	switch {
	case c.lastActivity+c.minPing < now:
		status = active
	case c.lastActivity+c.maxPing < now:
		status = disconnected
	default:
		status = notActive
	}
	return status
}

func (c *client) write(msgType int, msg []*message.Message) error {
	c.conn.SetWriteDeadline(time.Now().Add(time.Duration(c.rwdeadline) * time.Second))
	if msgType == websocket.PingMessage {
		return c.conn.WriteMessage(msgType, nil)
	}
	return c.conn.WriteJSON(msg)
}

func (c *client) WriteMessage(msg *message.Message) {
	c.msg <- msg
}

func (c *client) read() (*message.Message, error) {
	c.conn.SetReadDeadline(time.Now().Add(time.Duration(c.rwdeadline) * time.Second))
	var msg message.Message
	err := c.conn.ReadJSON(&msg)
	return &msg, err
}
