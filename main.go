package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/pschlafley/coding-challenges/go-memcache/server"
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

	server.HandleServerMessageQueue()

	log.Fatal(server.Start())
}
