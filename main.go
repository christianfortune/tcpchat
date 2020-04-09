package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
)

const serverAddr = ":8333"

// ChatClient is a chat client type
type ChatClient struct {
	Conn     net.Conn
	Address  string
	Username string
	Ok       bool
}

var currClients map[string]*ChatClient = map[string]*ChatClient{}

func main() {
	args := os.Args[1:]
	runType := args[0]
	switch runType {
	case "server":
		runServer()
	case "client":
		runClient()
	default:
		log.Fatalf("valid args are 'client' or 'server' only!")
	}
}

func runServer() {
	ln, err := net.Listen("tcp", serverAddr)
	if err != nil {
		log.Fatalf("could not setup listener: %v\n", err)
	}
	defer ln.Close()
	log.Printf("tcp server listening at %s", serverAddr)
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Fatalf("could not accept conn: %v\n", err)
		}
		defer conn.Close()
		clientAddr := conn.RemoteAddr().String()
		// Wait for username in first read
		buf := make([]byte, 1024)
		size, err := conn.Read(buf)
		if err != nil {
			if err == io.EOF {
				log.Printf("%s: [disconnected]", clientAddr)
				currClients[clientAddr].Ok = false
				return
			}
			log.Fatalf("could not read into buffer: %v\n", err)

		}
		username := string(buf[:size])
		cClient := &ChatClient{
			Ok:       true,
			Conn:     conn,
			Address:  clientAddr,
			Username: username,
		}
		currClients[clientAddr] = cClient
		log.Printf("%s: [connected]", clientAddr)
		broadcastMessage("[connected]", cClient)
		fmt.Fprintf(conn, "#### Welcome to TCPChat! ####")
		// waiting for any writes from client
		go func(conn net.Conn, c *ChatClient) {
			for {
				buf := make([]byte, 1024)
				size, err := c.Conn.Read(buf)
				if err != nil {
					if err == io.EOF {
						log.Printf("%s: [disconnected]\n", c.Address)
						broadcastMessage("[disconnected]", c)
						currClients[clientAddr].Ok = false
						return
					}
					log.Fatalf("could not read into buffer: %v\n", err)
				}
				out := string(buf[:size])
				log.Printf("%s: %s", conn.RemoteAddr(), out)
				broadcastMessage(out, c)
			}
		}(conn, cClient)
	}
}

func broadcastMessage(message string, from *ChatClient) {
	for addr, cc := range currClients {
		if addr == from.Address || !cc.Ok {
			continue
		}
		_, err := fmt.Fprintf(cc.Conn, "%s: %s", strings.Trim(from.Username, "\n"), message)
		if err != nil {
			log.Fatalf("could not write message to other client: %v\n", err)
		}
	}
}

func runClient() {
	conn, err := net.Dial("tcp", serverAddr)
	if err != nil {
		log.Fatalf("could not dial to tcp server: %v\n", err)
	}
	defer conn.Close()
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Printf("Please enter your username > ")
	scanner.Scan()
	username := scanner.Text()
	fmt.Fprintf(conn, "%s\n", username)
	// waiting for any responses
	go func() {
		for {
			buf := make([]byte, 1024)
			size, err := conn.Read(buf)
			if err != nil {
				log.Fatalf("could not read into buffer: %v\n", err)
			}
			data := buf[:size]
			fmt.Printf("%s", string(data))
		}
	}()
	for scanner.Scan() {
		text := scanner.Text()
		if text == "!exit" {
			os.Exit(1)
		}
		fmt.Fprintf(conn, "%s\n", text)
	}
	if err := scanner.Err(); err != nil {
		log.Println(err)
	}
}
