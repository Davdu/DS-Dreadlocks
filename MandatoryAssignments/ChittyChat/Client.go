package main

import (
	"bufio"
	"chittychat/chittychat"
	"context"
	"errors"
	"fmt"
	"google.golang.org/grpc"
	"log"
	"os"
	"strings"
	"unicode/utf8"
)

var lamportTime int32 = 1
var username string

func main() {
	fmt.Print("Please input a username: ")
	fmt.Scanln(&username)
	fmt.Printf("Username is: %s\n", username)

	// Create a logger, Courtesy of https://stackoverflow.com/questions/19965795/how-to-write-log-to-file
	f, err := os.OpenFile("LogFile", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0664)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer f.Close()
	log.SetOutput(f)

	// Connect to port
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	// Generate new client and initialize stream
	client := chittychat.NewChittyChatClient(conn)
	stream, err := client.Messages(context.Background())
	if err != nil {
		log.Fatalf("error while creating stream: %v", err)
	}

	// Send join message
	sendMessage(fmt.Sprintf("Client %s joined Chitty-Chat", username), stream)

	// Recieve messages on a different thread to not interfere with sending of messages.
	go receiveMessage(stream)

	// Infinite loop to detect newly written messages
	reader := bufio.NewReader(os.Stdin)
	for {
		text, _ := reader.ReadString('\n')
		sendMessage(strings.TrimSpace(text), stream)
	}
}

func incrementLamport() {
	lamportTime++
}

func sendMessage(message string, stream chittychat.ChittyChat_MessagesClient) {

	// Validates messages and prints and logs the given error message.
	err := validateMessage(message)
	if err != nil {
		fmt.Printf("%s\n", err)
		log.Printf("Invalid Message! - %s: %s\n", username, err)
		return
	}

	incrementLamport()
	msg := &chittychat.Message{
		Username:  username,
		Message:   message,
		Timestamp: lamportTime,
	}

	log.Printf("Client %s, Outgoing  - %s", username, msg)

	if err := stream.Send(msg); err != nil {
		fmt.Printf("Failed to send message: %v", err)
	}
}

func receiveMessage(stream chittychat.ChittyChat_MessagesClient) {
	for {
		incoming, err := stream.Recv()
		if err != nil {
			log.Fatalf("Failed to receive a message: %v", err)
			return
		}
		lamportTime = max(lamportTime, incoming.Timestamp)
		incrementLamport()

		log.Printf("Client %s, Incoming  - Username:\"%s\" Message:\"%s\" Timestamp:\"%d\"", username, incoming.Username, incoming.Message, lamportTime)
		fmt.Printf("%s: %s\n", incoming.Username, incoming.Message)
	}
}

func validateMessage(msg string) (err error) {

	if len(msg) > 128 {
		err = errors.New("message is too long. Keep it within 128 characters long")
		return
	}

	if !utf8.ValidString(msg) {
		err = errors.New("message is too long. Keep it within 128 characters long")
		return
	}

	return
}
