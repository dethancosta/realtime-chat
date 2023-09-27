package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/dethancosta/rtchat/utils"
	"github.com/redis/go-redis/v9"
)

const (
	HOST = "localhost"
	PORT = "7007"

	RedisAddress = "redis:6379"
	RedisChannel = "message" // Default channel
	RedisIdKey = "id"
	RedisMessagesKey = "messages"
)

var (
	redisClient = redis.NewClient(&redis.Options{
		Addr: RedisAddress,
		Password: "",
		DB: 0,
	})

	ctx = context.Background()
)

type ChatServer struct {
	connPool map[string]net.Conn
	ctx context.Context
	cancelFn context.CancelFunc
	sendChan chan string
	mux sync.RWMutex
}

func NewChatServer() *ChatServer {
	ctxt, cancelFunc := context.WithCancel(context.Background())

	return &ChatServer{
		connPool: make(map[string]net.Conn),
		ctx: ctxt,
		cancelFn: cancelFunc,
		sendChan: make(chan string),
		mux: sync.RWMutex{},
	}
}

func (cs *ChatServer) acceptMsgs(name string) {
	rdr := bufio.NewReader(cs.connPool[name])
	defer cs.connPool[name].Close()
	defer cs.cancelFn()

	for {
	select {
		case <-cs.ctx.Done():
			return

		default:
			msg, err := rdr.ReadBytes('\n')
			if err == io.EOF {
				log.Println(name + " left the chat.")
				cs.mux.Lock()
				delete(cs.connPool, name)
				cs.mux.Unlock()
				return
			} else if err != nil {
				log.Println(err.Error())
				return
			}

			var message utils.Message
			err = json.Unmarshal(msg, &message)
			if err != nil {
				log.Println(err.Error())
				return
			}

			message.Id, err = redisClient.Incr(ctx, RedisIdKey).Result()
			message.Date = time.Now()
			if err != nil {
				log.Println(fmt.Errorf("Can't increment redis id key: %w", err))
				return
			}

			data, err := json.Marshal(message)
			if err != nil {
				log.Println(fmt.Errorf("Can't increment redis id key: %w", err))
				return
			}
			if err := redisClient.RPush(ctx, RedisMessagesKey, data).Err(); err != nil {
				log.Printf("error pushing to redis: %v\n", err)
				return
			}

			if err := redisClient.Publish(ctx, RedisChannel, data).Err(); err != nil {
				log.Printf("error publishing to redis: %v\n", err)
				return
			}

			//cs.sendChan <-message.String()
		}
	}
}

func (cs *ChatServer) sendMsg() {
	defer cs.cancelFn()

	pubsub := redisClient.Subscribe(ctx, RedisChannel)
	for {
		select {
		case <-cs.ctx.Done():
			return
			/*
		case msg := <-cs.sendChan:
			cs.mux.RLock()
			for _, v := range cs.connPool {
				v.Write([]byte(msg))
			}
			cs.mux.RUnlock()
			*/
		default:
			msg, err := pubsub.ReceiveMessage(ctx)
			messages := make([]utils.Message, 1)
			var m2 utils.Message
			err = json.Unmarshal([]byte(msg.Payload), &m2)
			if err != nil {
				log.Printf("unmarshal in sendMsg: %v", err)
				return
			}

			messages[0] = m2
			jsonMsg, err := json.Marshal(messages) // TODO ensure slice is encoded correctly
			if err != nil {
				log.Printf("marshal in sendMsg: %v", err)
				return
			}

			cs.mux.RLock()
			for _, v := range cs.connPool {
				v.Write(jsonMsg)	
			}
			cs.mux.RUnlock()
		}
	}
}

func (cs *ChatServer) addConnection(conn net.Conn) {
	rdr := bufio.NewReader(conn)
	init, err := rdr.ReadString('\n')
	if err != nil {
		log.Println("Couldn't add connection: " + err.Error())
		return
	}

	name := strings.TrimSpace(init)
	cs.mux.Lock()
	cs.connPool[name] = conn
	cs.mux.Unlock()
	log.Println(name + " has entered the chat.")

	go cs.acceptMsgs(name)
}

func main() {
	listener, err := net.Listen("tcp", HOST + ":" + PORT)
	if err != nil {
		panic(err)
	}

	server := NewChatServer()
	go server.sendMsg()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println(err.Error())
			return
		}
		
		server.addConnection(conn)

	}
}
