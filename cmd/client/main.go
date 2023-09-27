package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"strings"

	"github.com/dethancosta/rtchat/utils"
)

const (
	HOST = "localhost"
	PORT = "7007"
)

func acceptInput(ctx context.Context, ch chan string, rdr *bufio.Reader) {
	for {

		select {
		case <-ctx.Done():
			return
		default:
			toSend, err := rdr.ReadString('\n')
			if err != nil {
				log.Println(err.Error())
				ch <- ""
				return
			}

			ch <- toSend
		}
	}
}

func handleMsg(ctx context.Context, ch chan string, rdr *bufio.Reader) {
	for {
		select {
		case <-ctx.Done():
			return
		default:

			recv, err := rdr.ReadString('\n')
			if err != nil {
				ch <- ""
				log.Println(err)
				return
			}
			recv = strings.TrimSpace(recv)

			ch <- recv
		}
	}
}

func main() {
	conn, err := net.Dial("tcp", HOST+":"+PORT)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
	defer conn.Close()
	sc := bufio.NewReader(os.Stdin)
	rdr := bufio.NewReader(conn)

	fmt.Print("name: ")
	name, err := sc.ReadString('\n')
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
	
	_, err = conn.Write([]byte(name))
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	name = strings.TrimSpace(name)

	ctx, cancelMsg := context.WithCancel(context.Background())
	defer cancelMsg() // TODO delete?

	sendCh := make(chan string)
	recvCh := make(chan string)

	go acceptInput(ctx, sendCh, sc)
	go handleMsg(ctx, recvCh, rdr)

	for {
		select {
		case s := <-sendCh:
			s = strings.TrimSpace(s)
			if s == "quit" || s == "exit" {
				cancelMsg()
				return
			}
			//s = name + ": " + s
			msg := utils.NewMessage()
			msg.Name = name
			msg.Content = s
			payload, err := json.Marshal(msg)
			if err != nil {
				log.Println(err)
				return
			}
			// fmt.Println(s)
			//_, err = conn.Write([]byte(s + "\n"))
			_, err = conn.Write(append(payload, '\n'))
			if err != nil {
				log.Println(err)
				return
			}
		case r := <-recvCh:
			fmt.Println(r)
		}
	}
}
