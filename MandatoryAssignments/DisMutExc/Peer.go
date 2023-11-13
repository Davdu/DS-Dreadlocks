package main

import (
	nd "DisMutExc/node"
	"bufio"
	"context"
	"fmt"
	"google.golang.org/grpc"
	"log"
	"net"
	"os"
)

type node struct {
	nd.UnimplementedNodeServer
	ID                        int32
	LamportTime               int32
	LamportTimeAtReq          int32
	State                     string
	Port                      string
	Peers                     []nd.Node_MessagesClient
	PeerIds                   map[int32]nd.Node_MessagesClient
	requestQueue              []*Request
	requestingCriticalSection bool
	repliesReceived           map[int32]bool
}

type Request struct {
	PeerID  int32
	Lamport int32
}

func (node node) mustEmbedUnimplementedNodeServer() {
	//TODO implement me
	panic("implement me")
}

func main() {

	// Create a logger, Courtesy of https://stackoverflow.com/questions/19965795/how-to-write-log-to-file
	f, err := os.OpenFile("LogFile", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0664)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer f.Close()
	log.SetOutput(f)

	fmt.Println("Please enter the Node ID")
	var nodeID int32
	_, err = fmt.Scan(&nodeID)

	node := &node{
		ID:               nodeID,
		LamportTime:      1,
		LamportTimeAtReq: 1,
		State:            "RELEASED",
		Port:             string(50000 + nodeID),
		Peers:            []nd.Node_MessagesClient{},
		PeerIds:          make(map[int32]nd.Node_MessagesClient),
	}

	// Start the server in a separate goroutine
	go startGRPCServer(node)

	establishConnections := false

	for !establishConnections {
		fmt.Println("Are you ready to establish connection to ALL other nodes?")
		_, err = fmt.Scan(&establishConnections)
	}

	// Establish connections to other nodes
	// You'll need to determine how to get the addresses of other nodes
	connectToOtherNodes(node)

	node.requestCriticalSection()
}

func startGRPCServer(node *node) {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", node.Port))
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	// Register your server implementation
	nd.RegisterNodeServer(grpcServer, node)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

func connectToOtherNodes(node *node) {
	for i := 0; i < 3; i++ {
		if !(node.ID == int32(i)) {
			openConnection(int32(i), node)
		}
	}
}

func openConnection(id int32, node *node) {

	var port = 50000 + id

	conn, err := grpc.Dial(string(port), grpc.WithInsecure())
	if err != nil {
		log.Printf(`Failed to connect to node at %s: %v`, port, err)
	}
	client := nd.NewNodeClient(conn)
	stream, err := client.Messages(context.Background())
	node.PeerIds[int32(id)] = stream
	node.Peers = append(node.Peers, stream)

	fmt.Println("Connected to port ", port)
}

func queue() {

}

func (node *node) reply(peerID int32) {
	if peer, exists := node.PeerIds[peerID]; exists {
		message := &nd.Message{
			Id:      node.ID,
			State:   "REPLY",
			Lamport: node.LamportTime,
		}
		if err := peer.Send(message); err != nil {
			log.Printf("Failed to send reply to peer %d: %v", peerID, err)
		}
	}
}

func (node *node) processRequestQueue() {
	for _, request := range node.requestQueue {
		node.reply(request.PeerID)
	}
	node.requestQueue = nil // Clear the queue
}

func (n *node) Messages(stream nd.Node_MessagesServer) error {
	for {
		incoming, err := stream.Recv()
		if err != nil {
			return err
		}

		n.LamportTime = max(n.LamportTime, incoming.Lamport) + 1

		switch incoming.State {
		case "WANTED":
			n.handleRequest(incoming)
		case "REPLY":
			n.repliesReceived[incoming.Id] = true
			if n.canEnterCriticalSection() {
				n.enterCriticalSection()
			}
		}
	}
}

func (n *node) handleRequest(incoming *nd.Message) {
	if n.State == "HELD" || (n.State == "WANTED" && n.isRequestPrior(incoming)) {
		n.requestQueue = append(n.requestQueue, &Request{
			PeerID:  incoming.Id,
			Lamport: incoming.Lamport,
		})
	} else {
		n.reply(incoming.Id)
	}
}

func (n *node) isRequestPrior(incoming *nd.Message) bool {
	return n.requestingCriticalSection && (incoming.Lamport < n.LamportTime || (incoming.Lamport == n.LamportTime && incoming.Id < n.ID))
}

func (n *node) canEnterCriticalSection() bool {
	for _, received := range n.repliesReceived {
		if !received {
			return false
		}
	}
	return true
}

func (n *node) enterCriticalSection() {
	n.State = "HELD"
	write(n)
	n.exitCriticalSection()
}

func (n *node) exitCriticalSection() {
	n.State = "RELEASED"
	n.requestingCriticalSection = false
	n.processRequestQueue()
}

func (node *node) broadcastMessage(msg *nd.Message) {
	//Increment lamport, maybe synchronize?

	// Loops through all clients and sends the given message.
	for _, client := range node.Peers {
		if err := client.Send(msg); err != nil {
			log.Printf("Failed to broadcast message: %v", err)
		}
	}
}

func (n *node) requestCriticalSection() {

	n.State = "WANTED"
	n.requestingCriticalSection = true
	n.repliesReceived = make(map[int32]bool)

	for _, peer := range n.Peers {
		message := &nd.Message{
			Id:      n.ID,
			State:   "WANTED",
			Lamport: n.LamportTime,
		}
		if err := peer.Send(message); err != nil {
			log.Printf("Failed to send critical section request: %v", err)
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

func (node *node) updateAndIncrementLamport(receivedTimestamp int32) int32 {
	node.LamportTime = max(node.LamportTime, receivedTimestamp) + 1
	return node.LamportTime
}

// Write methods hardcoded

func write(node *node) {
	data, err := os.OpenFile("DataFile", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0664)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer data.Close()

	w := bufio.NewWriter(data)
	_, err = w.WriteString("Hello World")
	if err != nil {
		return
	}

	log.Printf("Peer %v wrote to data at Lamport time %v.\n", node.ID, node.LamportTime)
}
