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
	startGRPCServer(server)

}

func startGRPCServer(server *AuctionServer) {
	lis, err := net.Listen("tcp", server.port)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	// Register your Server implementation
	s.RegisterAuctionServer(grpcServer, server)
	log.Printf("Server started on port %s", server.port)
	// Start listening for requests
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}

}

func establishConnections(server *AuctionServer) {
	for i := 0; ; i++ {

		err := openConnection(int32(i), server)
		if err != nil {
			server.ID = int32(i)
			server.port = fmt.Sprintf("localhost:%d", 50000+i)
			return
		}
	}
}

func openConnection(id int32, server *AuctionServer) error {
	var port = fmt.Sprintf("localhost:%d", 50000+id)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, port, grpc.WithInsecure(), grpc.WithBlock(), grpc.FailOnNonTempDialError(true))
	if err != nil {
		return err // Don't proceed if there's an error
	}

	client := s.NewAuctionClient(conn)
	var bs = &backupServer{Server: client, Conn: conn}

	server.backupServers = append(server.backupServers, bs)
	log.Println("Server: ", server.ID, " Connected to port ", port)
	return nil
}

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
			ack = &s.Ack{Valid: false, Comment: "Another leader exists"}
			return ack, nil
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
		synchronizeWithBackupServers(server)
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

func synchronizeWithBackupServers(server *AuctionServer) {
	server.updateAndIncrementLamport(server.lamportTime) // Update lamport time when sending a message
	for _, backupServer := range server.backupServers {
		ack, err := backupServer.Server.Sync(context.Background(), &s.Sync{
			HighestBid:      server.highestBid,
			HighestBidderId: server.highestBidderId,
			Sold:            server.sold,
			TimeRemaining:   server.timeRemaining,
		})
		if err != nil {
			return
		}
		if ack.Valid {
			// Do something with the ack
		}
	}
}
func (server *AuctionServer) checkOtherLeaderExists() bool {
	// Loops through backup servers and checks if any of them are the leader
	for _, backupServer := range server.backupServers {
		ack, err := backupServer.Server.IsLeader(context.Background(), &s.CheckLeader{Lamport: server.lamportTime})
		if err != nil {
			continue
		}
		if ack.Valid {
			return true
		}
	}
	return false
}

func (server *AuctionServer) updateAndIncrementLamport(receivedTimestamp int32) int32 {
	server.lamportTime = max(server.lamportTime, receivedTimestamp) + 1
	return server.lamportTime
}
