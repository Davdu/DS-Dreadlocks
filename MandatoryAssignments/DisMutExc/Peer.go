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
	"time"
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
	var nodeID int
	_, err = fmt.Scan(&nodeID)

	node := &node{
		ID:               int32(nodeID),
		LamportTime:      1,
		LamportTimeAtReq: 1,
		State:            "RELEASED",
		Port:             fmt.Sprintf("localhost:%d", 50000+nodeID),
		Peers:            []nd.Node_MessagesClient{},
		PeerIds:          make(map[int32]nd.Node_MessagesClient),
	}

	// Start the server in a separate goroutine
	go startGRPCServer(node)

	time.Sleep(10 * time.Second) // Establish connections to other nodes

	// You'll need to determine how to get the addresses of other nodes
	connectToOtherNodes(node)

	time.Sleep(5 * time.Second)

	go node.requestCriticalSection()

	time.Sleep(200000 * time.Second)
}

func startGRPCServer(node *node) {
	lis, err := net.Listen("tcp", node.Port)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	// Register your server implementation
	nd.RegisterNodeServer(grpcServer, node)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}

	log.Println("Peer: ", node.ID, " Started server")
}

func connectToOtherNodes(node *node) {
	for i := 0; i < 3; i++ {
		if !(node.ID == int32(i)) {
			openConnection(int32(i), node)
		}
	}
}

func openConnection(id int32, node *node) {
	var port = fmt.Sprintf("localhost:%d", 50000+id)
	conn, err := grpc.Dial(port, grpc.WithInsecure())
	if err != nil {
		log.Printf(`Peer: %v: Failed to connect to node at %s: %v`, node.ID, port, err)
		return // Don't proceed if there's an error
	}

	client := nd.NewNodeClient(conn)
	stream, err := client.Messages(context.Background())
	if err != nil {
		log.Printf(`Peer: %v: Failed to create stream with node at %s: %v`, node.ID, port, err)
		return // Don't proceed if there's an error
	}

	node.PeerIds[int32(id)] = stream
	node.Peers = append(node.Peers, stream)
	log.Println("Peer: ", node.ID, " Connected to port ", port)
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

		n.updateAndIncrementLamport(incoming.Lamport)

		log.Println("Peer: ", n.ID, " Recieved Message from client:", incoming.Id, " ", incoming.State, " at Lamport time: ", n.LamportTime)

		switch incoming.State {
		case "WANTED":
			n.handleRequest(incoming)
		case "REPLY":
			n.repliesReceived[incoming.Id] = true
			if n.canEnterCriticalSection() && n.requestingCriticalSection {
				n.requestingCriticalSection = false
				n.enterCriticalSection()
			}
		}
	}
}

func (n *node) handleRequest(incoming *nd.Message) {
	if n.State == "HELD" || (n.State == "WANTED" && n.isRequestPrior(incoming)) {
		n.updateAndIncrementLamport(n.LamportTime)
		n.requestQueue = append(n.requestQueue, &Request{
			PeerID:  incoming.Id,
			Lamport: incoming.Lamport,
		})
	} else {
		n.updateAndIncrementLamport(n.LamportTime)
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

	n.updateAndIncrementLamport(n.LamportTime)

	log.Printf("Peer: %v: Entering critical section at Lamport time %v", n.ID, n.LamportTime)

	n.State = "HELD"
	write(n)
	n.exitCriticalSection()
}

func (n *node) exitCriticalSection() {

	n.updateAndIncrementLamport(n.LamportTime)

	log.Printf("Peer: %v: Exiting critical section at Lamport time %v", n.ID, n.LamportTime)

	n.State = "RELEASED"
	n.processRequestQueue()
}

func (n *node) requestCriticalSection() {

	n.updateAndIncrementLamport(n.LamportTime)

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

	node.updateAndIncrementLamport(node.LamportTime)

	// Open the file with read, write, create and append mode
	data, err := os.OpenFile("DataFile", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0664)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer data.Close() // Ensure file is closed when function exits

	// Create a buffered writer
	w := bufio.NewWriter(data)

	// Write the string to the buffer

	_, err = w.WriteString("Hello World\n") // Added newline for readability in file
	if err != nil {
		log.Printf("Error writing to buffer: %v", err)
		return
	}

	// Flush the buffer to ensure data is written to the file
	err = w.Flush()
	if err != nil {
		log.Printf("Error flushing buffer to file: %v", err)
		return
	}

	// Log successful write
	log.Printf("Peer %v wrote to data at Lamport time %v.\n", node.ID, node.LamportTime)
}
