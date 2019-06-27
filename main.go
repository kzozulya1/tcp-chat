package main

import (
	"app/pkg/chat"
	"os"
)

//Main entry
func main() {
	port := os.Getenv("TCP_PORT")
	server := chat.NewServer()
	server.Listen(port)
}
