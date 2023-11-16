package main

import (
	s "Auction/Server"
	"context"
	"fmt"
	"google.golang.org/grpc"
	"log"
	"math/rand"
	"net"
	"os"
	"time"
)

type AuctionServer struct {
	s.UnimplementedAuctionServer
	ID              int32
	highestBid      int32
	highestBidderId int32
	sold            bool
	isLeader        bool
	port            string
	lamportTime     int32
	timeRemaining   int32
	backupServers   []*backupServer
}

type backupServer struct {
	Server s.AuctionClient
	Conn   *grpc.ClientConn
}

func main() {

	// Create a logger, Courtesy of https://stackoverflow.com/questions/19965795/how-to-write-log-to-file
	f, err := os.OpenFile("LogFile", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0664)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer f.Close()
	log.SetOutput(f)

	server := &AuctionServer{
		ID:              0,
		highestBid:      0,
		highestBidderId: 0,
		port:            fmt.Sprintf("localhost:%d", 50000),
		backupServers:   []*backupServer{},
	}

	establishConnections(server)
	go startGRPCServer(server)
	// Run Scenario
	scenario(server)
}

func scenario(server *AuctionServer) {
	for {
		// Insert possible scenario here
	}
}

//##################################################
//############# Connection Functions ###############
//##################################################

func startGRPCServer(server *AuctionServer) {
	lis, err := net.Listen("tcp", server.port)
	if err != nil {
		log.Printf("Failed to listen: %v\n", err)
		log.Println("New Server will retry connecting in 5 seconds")

		// This is a fallback for when this server attempts to serve on an occupied port.
		// It waits for 5 seconds and then looks for a new port to serve.
		time.Sleep(5 * time.Second)
		establishConnections(server)
		startGRPCServer(server)
		return
	}
	grpcServer := grpc.NewServer()
	// Register your Server implementation
	s.RegisterAuctionServer(grpcServer, server)
	log.Printf("Server: %v started on port %s", server.ID, server.port)
	// Start listening for requests
	go requestReturnConnections(server)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}

}

func establishConnections(server *AuctionServer) {
	// Tries to make a connection to port 50000 + i
	// If an error occurs, there doesn't exist a server on that port
	// ,and therefore it is available for this server to serve.

	for i := 0; ; i++ {
		err := openConnection(int32(i), server)
		if err != nil {
			server.ID = int32(i)
			server.port = fmt.Sprintf("localhost:%d", 50000+i)
			log.Printf("New server ID = %v, port = %v\n", server.ID, server.port)
			return
		}
	}
}

func openConnection(id int32, server *AuctionServer) error {

	// Determine port to try on
	var port = fmt.Sprintf("localhost:%d", 50000+id)

	// Make context with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Dial the port. If no error occurs, make a connection to dialed server,
	// otherwise, return and don't make a connection.
	conn, err := grpc.DialContext(ctx, port, grpc.WithInsecure(), grpc.WithBlock(), grpc.FailOnNonTempDialError(true))
	if err != nil {
		return err // Don't proceed if there's an error
	}

	// Append connection to backupServers slice
	otherServer := s.NewAuctionClient(conn)
	var bs = &backupServer{Server: otherServer, Conn: conn}
	server.backupServers = append(server.backupServers, bs)
	server.updateAndIncrementLamport(server.lamportTime) // Update lamport when connection has opened
	fmt.Printf("New connection: Connected to port %v, at Lamport: %v\n", port, server.lamportTime)

	return nil
}

func requestReturnConnections(server *AuctionServer) {

	time.Sleep(5 * time.Second) // Wait until this server is likely online

	server.updateAndIncrementLamport(server.lamportTime) // Update when sending a message
	for _, backupServer := range server.backupServers {
		ack, err := backupServer.Server.ReturnConnection(context.Background(), &s.ReturnConnection{
			ID:      server.ID,
			Lamport: server.lamportTime,
		})
		server.updateAndIncrementLamport(ack.Lamport) // Update when receiving a message
		if err != nil {
			continue
		}
	}
}

//##################################################
//############### RPC responders ###################
//##################################################

func (server *AuctionServer) Sync(ctx context.Context, incoming *s.Sync) (*s.Ack, error) {

	server.updateAndIncrementLamport(incoming.Lamport) // Update lamport time when receiving a message

	server.highestBid = incoming.HighestBid
	server.highestBidderId = incoming.HighestBidderId
	server.sold = incoming.Sold
	server.timeRemaining = incoming.TimeRemaining

	server.updateAndIncrementLamport(incoming.Lamport) // Update lamport time when sending a message
	return &s.Ack{Valid: true}, nil
}

func (server *AuctionServer) Bid(ctx context.Context, incoming *s.Bid) (ack *s.Ack, err error) {

	server.updateAndIncrementLamport(incoming.Lamport) // Update lamport time when receiving a message

	// Ensure that the server is the leader
	if !server.isLeader {
		// Check all other servers for an active leader
		if server.checkOtherLeaderExists() {
			server.updateAndIncrementLamport(server.lamportTime) // Update when sending a message
			ack = &s.Ack{Valid: false, Comment: "Another leader exists"}
			return ack, nil // don't handle bids
		} else {
			server.isLeader = true
		}
	}

	// Handle bid
	if server.highestBid > incoming.Amount {
		ack = &s.Ack{Valid: false, Comment: "Amount too low. Increase bid to at least " + string(server.highestBid+1)}

	} else if server.highestBidderId == incoming.ID {
		ack = &s.Ack{Valid: false, Comment: "You are already the highest bidder"}

	} else if server.sold {
		ack = &s.Ack{Valid: false, Comment: "Item has already been sold"}

	} else {
		server.highestBid = incoming.Amount
		server.highestBidderId = incoming.ID
		server.synchronizeWithBackupServers()
		ack = &s.Ack{Valid: true}
	}

	// Round off
	server.timeRemaining -= rand.Int31n(10)            // Randomly decrement time remaining to simulate time passing
	server.updateAndIncrementLamport(incoming.Lamport) // Update lamport time when sending a message
	ack.Lamport = server.lamportTime                   // set ack lamport time to the current lamport time after updating it
	return ack, nil

}

func (server *AuctionServer) IsLeader(ctx context.Context, incoming *s.CheckLeader) (ack *s.Ack, err error) {

	server.updateAndIncrementLamport(incoming.Lamport) // Update lamport time when receiving a message

	if server.isLeader {
		ack = &s.Ack{Valid: true}
	} else {
		ack = &s.Ack{Valid: false}
	}

	server.updateAndIncrementLamport(incoming.Lamport) // Update lamport time when sending a message
	return ack, nil

}

func (server *AuctionServer) ReturnConnection(ctx context.Context, incoming *s.ReturnConnection) (ack *s.Ack, err error) {

	server.updateAndIncrementLamport(incoming.Lamport) // Update lamport time  when receiving a message

	err = openConnection(incoming.ID, server)
	if err != nil {
		ack = &s.Ack{Valid: false, Comment: "Connection failed", Lamport: server.lamportTime}
	} else {
		ack = &s.Ack{Valid: true, Comment: "Connection established", Lamport: server.lamportTime}
	}
	server.updateAndIncrementLamport(server.lamportTime) // Update lamport time when sending a message
	ack.Lamport = server.lamportTime
	return
}

//##################################################
//############## RPC callers #######################
//##################################################

func (server *AuctionServer) synchronizeWithBackupServers() {
	server.updateAndIncrementLamport(server.lamportTime) // Update for each message sent
	for _, backupServer := range server.backupServers {
		ack, err := backupServer.Server.Sync(context.Background(), &s.Sync{
			HighestBid:      server.highestBid,
			HighestBidderId: server.highestBidderId,
			Sold:            server.sold,
			TimeRemaining:   server.timeRemaining,
			Lamport:         server.lamportTime,
		})
		if err != nil {
			return
		}
		server.updateAndIncrementLamport(ack.Lamport) // Update for each lamport received
		if ack.Valid {
			// Do something with the ack
		}
	}
}

func (server *AuctionServer) checkOtherLeaderExists() bool {
	// Loops through backup servers and checks if any of them are the leader
	server.updateAndIncrementLamport(server.lamportTime) // Update for each message sent
	for _, backupServer := range server.backupServers {
		ack, err := backupServer.Server.IsLeader(context.Background(), &s.CheckLeader{Lamport: server.lamportTime})
		if err != nil {
			continue
		}
		server.updateAndIncrementLamport(ack.Lamport) // Update for each message received
		if ack.Valid {
			return true
		}
	}
	return false
}

//##################################################
//############## Internal functions ################
//##################################################

func (server *AuctionServer) updateAndIncrementLamport(receivedTimestamp int32) int32 {
	server.lamportTime = max(server.lamportTime, receivedTimestamp) + 1
	return server.lamportTime
}
