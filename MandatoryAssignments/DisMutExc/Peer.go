package main

import (
	nd "DisMutExc/node"
	"context"
	"fmt"
	"google.golang.org/grpc"
	"log"
	"os"
)

type node struct {
	ID               int32
	LamportTime      int32
	LamportTimeAtReq int32
	State            string
	peers            []nd.Node_MessagesClient
	peerIds          map[int32]nd.Node_MessagesClient
}

func main() {

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

	initializeClient(conn)
}

func initializeClient(conn *grpc.ClientConn) {

	client := nd.NewNodeClient(conn)
	stream, err := client.Messages(context.Background())
	if err != nil {
		log.Fatalf("error while creating stream: %v", err)
	}

}

func queue() {

}

func reply(node *node, peer nd.Node_MessagesClient) {

	message := &nd.Message{
		Id:      node.ID,
		State:   node.State,
		Lamport: node.LamportTime,
	}

	err := peer.Send(message)
	if err != nil {
		fmt.Println("Could not send message from client ID: ", node.ID)
	}
}

func (node *node) Messages(stream nd.Node_MessagesClient) error {

	node.peers = append(node.peers, stream)

	for {

		incoming, err := stream.Recv()

		node.peerIds[incoming.Id] = stream

		if err != nil {
			return err
		}

		if !(node.State == "RELEASED") {

			peer := node.peerIds[incoming.Id]

			reply(node, peer)
		} else {
			if incoming.State == "HELD" ||
				incoming.State == "WANTED" &&
					node.LamportTimeAtReq < incoming.Lamport-1 {

				queue(incoming)

			} else {

			}
		}

		incoming.Lamport = node.updateAndIncrementLamport(incoming.Timestamp)
		//node.broadcastMessage(incoming)
	}
}

func (node *node) broadcastMessage(msg *nd.Message) {
	//Increment lamport, maybe synchronize?
	//msg.Timestamp =
	log.Printf("Server, Outgoing  -  %node", msg)
	// Loops through all clients and sends the given message.
	for _, client := range node.peers {
		if err := client.Send(msg); err != nil {
			log.Printf("Failed to broadcast message: %v", err)
		}
	}
}

// Compares and returns the largest int32.
func max(a, b int32) int32 {
	if a > b {
		return a
	}
	return b
}
