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
	ID            int32
	highestBid    int32
	highestBidder string
	sold          bool
	isLeader      bool
	port          string
	timeRemaining int32
	backupServers []*backupServer
}

type backupServer struct {
	Server s.AuctionClient
	Conn   *grpc.ClientConn
}

func main() {
	// Create a logger and open a log file
	f, err := os.OpenFile("LogFile", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0664)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			log.Fatalf("error closing file: %v", err)
		}
	}(f)
	log.SetOutput(f)

	// Create an AuctionServer instance with initial parameters
	server := &AuctionServer{
		ID:            0,
		highestBid:    0,
		highestBidder: "",
		port:          fmt.Sprintf("localhost:%d", 50000),
		backupServers: []*backupServer{},
		timeRemaining: 50,
	}

	// Establish connections to other servers and reserve a port
	establishConnectionsAndReservePort(server)

	// Start the gRPC server
	startGRPCServer(server)
}

//##################################################
//############# Connection Functions ###############
//##################################################

// startGRPCServer initializes the gRPC server for the AuctionServer instance.
// It attempts to listen on the specified port, and if unsuccessful, retries after a 5-second delay.
func startGRPCServer(server *AuctionServer) {
	// Attempt to listen on the specified port
	lis, err := net.Listen("tcp", server.port)
	if err != nil {
		// Log the failure to listen
		log.Printf("Failed to listen: %v\n", err)
		log.Printf("Server --: Port already taken. Likely caused by multiple servers reserving same port before one starts.\n " +
			"Will reserve new port in 5 seconds...\n")

		// Wait for 5 seconds before retrying and looking for a new port to allow for another server to finish starting
		time.Sleep(5 * time.Second)
		establishConnectionsAndReservePort(server)
		startGRPCServer(server)
		return
	}

	// Create a new gRPC server
	grpcServer := grpc.NewServer()

	// Register the AuctionServer implementation
	s.RegisterAuctionServer(grpcServer, server)

	// Start listening for incoming requests
	go requestReturnConnections(server)

	// Serve gRPC requests
	log.Printf("Server %v: Now serving requests on %v\n", server.ID, server.port)
	fmt.Printf("Server %v: Now serving requests on %v\n", server.ID, server.port)

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Server %v: Failed to serve: %v", server.ID, err)
	}
}

// establishConnectionsAndReservePort attempts to establish connections to servers on consecutive ports
// starting from 50000. It reserves a range of 10 consecutive free ports for the current server.
func establishConnectionsAndReservePort(server *AuctionServer) {
	// Tries to make a connection to port 50000 + i
	// If an error occurs, there doesn't exist a server on that port
	// ,and therefore it is available for this server to serve.
	// While iterating through the ports, it also tries to make a connection to
	// the server on that port. It then continues iterating until it has found
	// 10 free ports in a row.

	portSet := false
	freePortsInARow := 0

	// Reset the backupServers slice
	server.backupServers = []*backupServer{}

	for i := 0; ; i++ {
		// Attempt to open a connection to the server on port 50000 + i
		err := openConnection(int32(i), server)
		if err != nil {
			// Increment the count of consecutive free ports
			freePortsInARow++

			// If the port is not already set, set it to the current free port
			if !portSet {
				server.ID = int32(i)
				server.port = fmt.Sprintf("localhost:%d", 50000+i)
				portSet = true
			} else if freePortsInARow >= 10 {
				// If 10 consecutive free ports are found, return
				break
			}
		} else {
			// Reset the count of consecutive free ports if a connection is successfully established
			freePortsInARow = 0
		}
	}

	for i := range server.backupServers {
		log.Printf("Server %v: Connected to server on port %v\n", server.ID, server.backupServers[i].Conn.Target())
	}

}

// canConnectToPort checks if a connection can be established to the specified port.
// It returns a boolean indicating the connection status, the gRPC client connection, and any error encountered.
func canConnectToPort(port string) (bool, *grpc.ClientConn, error) {
	// Make context with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Dial the port. If no error occurs, make a connection to the dialed server,
	// otherwise, return without making a connection.
	conn, err := grpc.DialContext(ctx, port, grpc.WithInsecure(), grpc.WithBlock(), grpc.FailOnNonTempDialError(true))
	if err != nil {
		return false, nil, err // Don't proceed if there's an error
	}

	return true, conn, nil
}

// openConnection attempts to open a connection to the server on the specified port.
// It checks if a connection can be established, and if successful, appends the connection to the backupServers slice.
func openConnection(id int32, server *AuctionServer) error {
	// Determine port to try on
	var port = fmt.Sprintf("localhost:%d", 50000+id)

	// Check if a connection can be established to the specified port
	_, conn, err := canConnectToPort(port)
	if err != nil {
		return err
	}

	// Create a new gRPC client based on the obtained connection
	otherServer := s.NewAuctionClient(conn)

	// Append the connection to the backupServers slice
	var bs = &backupServer{Server: otherServer, Conn: conn}
	server.backupServers = append(server.backupServers, bs)

	fmt.Printf("Connected to server on port %v\n", conn.Target())

	return nil
}

// removeUnavailableServers checks the availability of backup servers and removes the unavailable ones from the backupServers slice.
func removeUnavailableServers(server *AuctionServer) {
	// Checks if all backup servers are available
	// If not, remove them from the backupServers slice
	for i := len(server.backupServers) - 1; i >= 0; i-- {
		// Check if a connection can be established to the target of the current backup server
		_, _, err := canConnectToPort(server.backupServers[i].Conn.Target())
		if err != nil {
			// Remove connection from backupServers slice
			log.Printf("Server %v: Connection to server %v lost\n", server.ID, server.backupServers[i].Conn.Target())

			// Swap the element to the end of the slice
			server.backupServers[i], server.backupServers[len(server.backupServers)-1] = server.backupServers[len(server.backupServers)-1], server.backupServers[i]

			// Truncate the slice
			server.backupServers = server.backupServers[:len(server.backupServers)-1]
		}
	}
}

// requestReturnConnections sends a ReturnConnection request to each backup server in the server's backupServers slice.
func requestReturnConnections(server *AuctionServer) {
	// Wait until this server is likely online
	time.Sleep(5 * time.Second)

	// Iterate through backup servers and send a ReturnConnection request
	for _, backupServer := range server.backupServers {
		// Send a ReturnConnection request to the backup server
		_, err := backupServer.Server.ReturnConnection(context.Background(), &s.ReturnConnection{ID: server.ID})

		// Continue to the next backup server in case of an error
		if err != nil {
			continue
		}
	}
}

//##################################################
//############### RPC responders ###################
//##################################################

// Result handles the CallForUpdate RPC call, providing the current state of the auction server to the client.
func (server *AuctionServer) Result(_ context.Context, _ *s.Empty) (*s.Sync, error) {

	// Return a Sync message with the current state of the server
	return &s.Sync{
		HighestBid:    server.highestBid,
		HighestBidder: server.highestBidder,
		Sold:          server.sold,
		TimeRemaining: server.timeRemaining,
	}, nil
}

// Sync handles the Sync RPC call, synchronizing the state of the auction server with the incoming Sync message.
func (server *AuctionServer) Sync(_ context.Context, incoming *s.Sync) (*s.Ack, error) {

	// Update server state with the incoming Sync message
	server.highestBid = incoming.HighestBid
	server.highestBidder = incoming.HighestBidder
	server.sold = incoming.Sold
	server.timeRemaining = incoming.TimeRemaining

	log.Printf("Server %v: Synchronized\n", server.ID)

	// Return a valid acknowledgment
	return &s.Ack{Valid: true}, nil
}

// Bid handles the Bid RPC call, processing incoming bids and updating the server state accordingly.
func (server *AuctionServer) Bid(_ context.Context, incoming *s.Bid) (ack *s.Ack, err error) {

	// Ensure that the server is the leader
	if !server.isLeader {
		// Check all other servers for an active leader
		if server.checkOtherLeaderExists() {
			// Another server is the leader, return acknowledgment with appropriate ReturnCode
			ack = &s.Ack{Valid: false, ReturnCode: 1} // Another server is the leader
			return ack, nil                           // Don't handle bids
		} else {
			server.isLeader = true
			log.Printf("Server: %v is now the leader\n", server.ID)
		}
	}

	log.Printf("Server %v: Received bid from %v for %v\n", server.ID, incoming.ID, incoming.Amount)

	// Handle bid
	if server.highestBid >= incoming.Amount {
		ack = &s.Ack{Valid: false, ReturnCode: 2} // Bid too low. Increase bid above the highest bid.

	} else if server.highestBidder == incoming.ID {
		ack = &s.Ack{Valid: false, ReturnCode: 3} // You are already the highest bidder

	} else if server.sold {
		ack = &s.Ack{Valid: false, ReturnCode: 4} // Item has already been sold

	} else {
		// Update server state with the incoming bid
		server.highestBid = incoming.Amount
		server.highestBidder = incoming.ID
		log.Printf("Server %v: Highest bid is now %v from %v\n", server.ID, server.highestBid, server.highestBidder)
		ack = &s.Ack{Valid: true, ReturnCode: 0} // Bid successful
	}

	server.timeRemaining -= rand.Int31n(10) // Randomly decrement time remaining to simulate time passing
	if server.timeRemaining <= 0 {          // Check if the time has expired, mark the item as sold
		server.sold = true
		log.Printf("Server %v: Item sold to %v for %v\n", server.ID, server.highestBidder, server.highestBid)
	}

	// Round off
	server.synchronizeBackupServers() // Synchronize with backup servers

	if ack.ReturnCode != 0 {
		log.Printf("Server %v: Bid unsuccessful\n", server.ID)
	}

	return ack, nil
}

// IsLeader handles the CheckLeader RPC call, determining if the server is the leader.
func (server *AuctionServer) IsLeader(_ context.Context, _ *s.Empty) (ack *s.Ack, err error) {

	// Check if the server is the leader
	if server.isLeader {
		ack = &s.Ack{Valid: true}
	} else {
		ack = &s.Ack{Valid: false}
	}

	return ack, nil
}

// ReturnConnection handles the ReturnConnection RPC call, allowing other servers to reconnect.
func (server *AuctionServer) ReturnConnection(_ context.Context, incoming *s.ReturnConnection) (ack *s.Ack, err error) {

	// Attempt to open a connection to the server with the given ID
	err = openConnection(incoming.ID, server)
	if err != nil {
		// Connection failed, return acknowledgment with appropriate ReturnCode
		ack = &s.Ack{Valid: false, ReturnCode: 1}
		log.Printf("Server %v: Failed to connect to server on port %v\n", server.ID, "localhost:"+string(50000+incoming.ID))
	} else {
		// Connection successful, return acknowledgment with appropriate ReturnCode
		ack = &s.Ack{Valid: true, ReturnCode: 0}
		log.Printf("Server %v: Connected to server on port %v\n", server.ID, server.backupServers[len(server.backupServers)-1].Conn.Target())
	}

	return ack, nil
}

//##################################################
//############## RPC callers #######################
//##################################################

// synchronizeBackupServers sends synchronization messages to backup servers.
func (server *AuctionServer) synchronizeBackupServers() {

	errorOccurred := false

	// Iterate over backup servers and send synchronization messages
	for _, backupServer := range server.backupServers {
		// Send Sync message to backup server
		ack, err := backupServer.Server.Sync(context.Background(), &s.Sync{
			HighestBid:    server.highestBid,
			HighestBidder: server.highestBidder,
			Sold:          server.sold,
			TimeRemaining: server.timeRemaining,
		})

		// Check for errors in sending message
		if err != nil {
			errorOccurred = true
			continue
		}

		// Process the acknowledgment if it's valid
		if ack.Valid {
			log.Printf("Server %v: Synchronized server on port %v\n", server.ID, backupServer.Conn.Target())
		}
	}

	// If any errors occurred, remove unavailable servers
	if errorOccurred {
		removeUnavailableServers(server)
	}
}

// checkOtherLeaderExists checks if any of the backup servers is the leader.
func (server *AuctionServer) checkOtherLeaderExists() bool {
	errorOccurred, otherLeaderExists := false, false

	// Iterate over backup servers and check for a leader
	for _, backupServer := range server.backupServers {
		// Send IsLeader message to backup server
		ack, err := backupServer.Server.IsLeader(context.Background(), &s.Empty{})

		// Check for errors in sending message
		if err != nil {
			errorOccurred = true
			continue
		}

		// If the acknowledgment indicates that the other server is the leader, set the flag
		if ack.Valid {
			otherLeaderExists = true
		}
	}

	// If any errors occurred, remove unavailable servers
	if errorOccurred {
		removeUnavailableServers(server)
	}

	return otherLeaderExists
}
