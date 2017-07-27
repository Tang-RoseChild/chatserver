package chat

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/Tang-RoseChild/chatserver/message"

	"github.com/go-redis/redis"
)

const (
	onlineCountKey = "proj:online-count" // hash
)

type Hub struct {
	redisClient *redis.Client
	Name        string
	broadcast   chan *message.Message
	clients     map[*client]struct{}
	pub         chan *message.Message
	channelName string
	mux         sync.Mutex
	worker      chan *job
	done        chan struct{}
}

type job struct {
	c   *client
	msg *message.Message
}

const (
	hubDefaultChanSize       = 20
	hubDefaultMaxPubPerFlush = 20
)

func NewHub(name string, redisClient *redis.Client, channel string) *Hub {
	return &Hub{
		redisClient: redisClient,
		Name:        name,
		broadcast:   make(chan *message.Message, hubDefaultChanSize),
		clients:     make(map[*client]struct{}),
		pub:         make(chan *message.Message, hubDefaultChanSize),
		channelName: channel,
		worker:      make(chan *job, 3*hubDefaultChanSize),
		done:        make(chan struct{}),
	}
}

func (h *Hub) Run() {
	sub := h.redisClient.Subscribe(h.channelName)
	defer func() {
		close(h.done)
		sub.Close()
		h.redisClient.HDel(onlineCountKey, h.Name)
	}()
	go func() {
		var messages []*message.Message
		for {
			redisMsg, err := sub.ReceiveMessage()
			if err != nil {
				log.Println(err)
				continue
			}

			err = json.Unmarshal([]byte(redisMsg.Payload), &messages)
			if err != nil {
				log.Println(err)
				continue
			}
			for _, m := range messages {
				h.broadcast <- m
			}
		}
	}()

	go h.broadcastOnlineCount()
	go h.messageHandler()

	pubTicker := time.NewTicker(time.Millisecond * 10)
	msgs := make([]*message.Message, hubDefaultMaxPubPerFlush)
	for {
		select {
		case <-pubTicker.C:
			n := len(h.pub)
			if n == 0 {
				continue
			}
			if n > hubDefaultMaxPubPerFlush {
				n = hubDefaultMaxPubPerFlush
			}

			for i := 0; i < n; i++ {
				msgs[i] = <-h.pub
			}

			data, _ := json.Marshal(msgs[:n])
			h.redisClient.Publish(h.channelName, string(data)).Val()
		}

	}
}

func (h *Hub) broadcastOnlineCount() {
	h.redisClient.HSet(onlineCountKey, h.Name, 0)
	ticker := time.NewTicker(time.Second)
	for {
		select {
		case <-h.done:
			return
		case <-ticker.C:
			h.mux.Lock()
			length := len(h.clients)
			h.mux.Unlock()
			tx := h.redisClient.TxPipeline()
			tx.HSet(onlineCountKey, h.Name, length)
			var counts []int
			// tx.HVals(onlineCountKey).ScanSlice(&counts)
			strCmds := tx.HVals(onlineCountKey)
			if _, err := tx.Exec(); err != nil {
				log.Println(err)
				continue
			}
			strCmds.ScanSlice(&counts)
			var count int
			for _, v := range counts {
				count += v
			}

			h.broadcast <- &message.Message{
				Type:    message.MsgTypeInfo,
				Payload: &message.Payload{InfoMessage: &message.InfoMessage{OnlineCount: count}}}
		}
	}
}

func (h *Hub) messageHandler() {
	for i := 0; i < 3*hubDefaultChanSize; i++ {

		go func(worker chan *job) {
			for {
				select {
				case <-h.done:
					return
				case _job := <-worker:
					_job.c.WriteMessage(_job.msg)
				}
			}
		}(h.worker)
	}
	for {
		select {
		case <-h.done:
			return
		case msg, ok := <-h.broadcast:
			if !ok {
				return
			}
			for c := range h.clients { // may race?
				h.worker <- &job{c: c, msg: msg}
			}
		}
	}
}

func (h *Hub) Broadcast(msg *message.Message) {
	h.broadcast <- msg
}

func (h *Hub) Pub(msg *message.Message) {
	h.pub <- msg
}

func (h *Hub) Join(c *client) {
	h.mux.Lock()
	h.clients[c] = struct{}{}
	h.mux.Unlock()
}

func (h *Hub) Leave(c *client) {
	h.mux.Lock()
	if _, ok := h.clients[c]; ok {
		delete(h.clients, c)
	}
	h.mux.Unlock()
}
