package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/pschlafley/coding-challenges/go-memcache/server"
	"github.com/pschlafley/coding-challenges/go-memcache/types"
)

func main() {
	var portFlag string

	flag.StringVar(&portFlag, "p", "11211", "Enter in the port you want to bind the tcp server to")

	flag.Parse()

	if portFlag == "11211" {
		fmt.Println("starting on default port")
	} else {
		fmt.Println("starting on port" + portFlag)
	}

	address := "127.0.0.1:" + portFlag

	server := server.NewServer(address)

	go func() {
		messageQueue := types.NewQueue[types.Message]()
		for {
			msg := <-server.MsgCh
			messageQueue.Enque(msg)

			node := messageQueue.Head()

			for node != nil {
				fmt.Printf("%v %s: %s\n", node.Value().TimeStamp, node.Value().RemoteAddr, node.Value().Text)
				node = node.Next()
			}
		}

	}()

	log.Fatal(server.Start())
}
