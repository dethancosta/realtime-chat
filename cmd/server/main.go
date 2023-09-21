package main

import (
	"bufio"
	"context"
	"io"
	"log"
	"net"
	"strings"
	"sync"
)

const (
	HOST = "localhost"
	PORT = "7007"
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
	for {
	select {
		case <-cs.ctx.Done():
			return
		default:
			msg, err := rdr.ReadString('\n')
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
			cs.sendChan <-msg
		}
	}
}

func (cs *ChatServer) sendMsg() {
	for {
		select {
		case <-cs.ctx.Done():
			return
		case msg := <-cs.sendChan:
			cs.mux.RLock()
			for _, v := range cs.connPool {
				v.Write([]byte(msg))
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
