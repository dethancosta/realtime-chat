package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
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
	defer cancelMsg()
	for {

		select {
		case <-ctx.Done():
			return
		default:
			toSend, err := rdr.ReadString('\n')
			if err != nil {
				log.Println(fmt.Errorf("acceptInput: %w", err))
				ch <- ""
				return
			}

			ch <- toSend
		}
	}
}

func formatMessage(msg utils.Message) string {
	st := "----- " + msg.Name + " ----- " + msg.Date.String() + "\n"
	st += msg.Content + "\n" + strings.Repeat("-", len(st))
	return st
}

func handleMsg(ctx context.Context, ch chan string, rdr *bufio.Reader) {
	defer cancelMsg()
	for {
		select {
		case <-ctx.Done():
			return
		default:

			recv, err := rdr.ReadString('\n')
			if err == io.EOF {
				ch <- ""
				log.Println("handleMsg: EOF")
				return
			}
			if err != nil {
				ch <- ""
				log.Println(fmt.Errorf("handleMsg: %w", err))
				return
			}

			recv = strings.TrimSpace(recv)
			var msgArray []utils.Message

			err = json.Unmarshal([]byte(recv), &msgArray)
			if err != nil {
				ch <- ""
				log.Println("error unmarshalling: " + err.Error())
				return
			}

			for m := range msgArray {
				ch <- formatMessage(msgArray[m])
			}
		}
	}
}

var (
	ctx, cancelMsg = context.WithCancel(context.Background())
	name string
)

func main() {
	conn, err := net.Dial("tcp", HOST+":"+PORT)
	if err != nil {
		log.Fatal(fmt.Errorf("couldn't establish connection: %w", err))
		os.Exit(1)
	}
	defer func() {
		err = conn.Close()
		if err != nil {
			log.Printf("error closing connection: %s\n", err.Error())
		}
	}()
	defer cancelMsg()

	sc := bufio.NewReader(os.Stdin)
	rdr := bufio.NewReader(conn)

	fmt.Print("name: ")
	name, err = sc.ReadString('\n')
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
	
	_, err = conn.Write([]byte(name))
	if err != nil {
		log.Fatal(fmt.Errorf("couldn't write to conn: %w", err))
		os.Exit(1)
	}

	name = strings.TrimSpace(name)

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

			msg := utils.NewMessage()
			msg.Name = name
			msg.Content = s
			payload, err := json.Marshal(msg)
			if err != nil {
				log.Println(fmt.Errorf("main marshal: %w", err))
				return
			}

			_, err = conn.Write(append(payload, '\n'))
			if err != nil {
				log.Println(fmt.Errorf("conn.Write in main: %w", err))
				return
			}
		case r := <-recvCh:
			fmt.Println(r)
		}
	}
}
