package main

import (
	s "Auction/Service"
	"context"
	"fmt"
	"google.golang.org/grpc"
	"log"
	"os"
	"time"
)

type AuctionClient struct {
	s.UnimplementedAuctionServer
	server        s.AuctionClient
	ID            string
	HighestBid    int32
	HighestBidder string
}

func main() {
	// Create a logger and open a log file
	f, err := os.OpenFile("LogFile", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0664)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer f.Close()
	log.SetOutput(f)

	// Prompt the user to enter the client name
	var ID string
	fmt.Println("Please enter the client name")
	_, err = fmt.Scan(&ID)

	// Create a new AuctionClient instance with the provided client name
	client := &AuctionClient{
		ID: ID,
	}

	// Connect to the server and look for a leader
	client.connectToServer(true)

	// Update client information
	client.Update()

	// Start a goroutine to update client information intermittently
	go client.updateIntermittently()

	// Begin the process of taking bids
	client.takeBids()
}

// connectToServer attempts to establish a connection to a valid server.
// If 'lookForLeader' is true, it ensures that the connected server is also the leader.
func (client *AuctionClient) connectToServer(lookForLeader bool) {
	// Connect to a valid server using the specified criteria
	conn := client.findValidServerConn(lookForLeader)

	// Handle the case where no valid connection is obtained
	if conn == nil {
		fmt.Println("Could not connect to server. Restart client and try again.")
		log.Fatalf("Client %v: Could not connect to server\n", client.ID)
	}

	// Log and print the successful connection information
	log.Printf("Client %v: Connected to server on %v\n", client.ID, conn.Target())
	fmt.Printf("Connected to server on %v\n", conn.Target())

	// Create a gRPC client based on the obtained connection
	client.server = s.NewAuctionClient(conn)
}

// findValidServerConn attempts to establish a connection to a valid server within a given timeout.
// If 'lookForLeader' is true, it ensures that the server found is also the leader.
func (client *AuctionClient) findValidServerConn(lookForLeader bool) (connection *grpc.ClientConn) {
	// Set a timeout for the connection attempt
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	invalidPortsInARow := 0

	// Iterate until 10 consecutive invalid ports are found or a valid connection is established
	// If not looking for a leader, the client will connect to the first server it finds.
	// If looking for a leader, the client will connect to the first server it finds that is also the leader.
	for i := 0; ; i++ {

		if invalidPortsInARow >= 10 {
			// If 10 consecutive invalid ports are found and 'lookForLeader' is true, break out of the loop
			break
		}

		// Generate the port based on the iteration
		port := fmt.Sprintf("localhost:%d", 50000+i)

		// Attempt to establish a gRPC connection to the server
		conn, err := grpc.DialContext(ctx, port, grpc.WithInsecure(), grpc.WithBlock(), grpc.FailOnNonTempDialError(true))
		if err != nil {
			// If there is an error, indicating no server available on port, continue to the next iteration
			invalidPortsInARow++
			continue
		} else if lookForLeader {
			// If 'lookForLeader' is true, check if the connected server is the leader

			invalidPortsInARow = 0 // Reset the counter because a valid port was found

			// If looking for a leader, check if the connected server is the leader
			ack, err := s.NewAuctionClient(conn).IsLeader(context.Background(), &s.Empty{})
			if err != nil {
				// If there is an error, continue to the next iteration
				invalidPortsInARow++
				continue
			}

			if ack.Valid {
				// If the connected server is the leader, set 'connection' and break out of the loop
				connection = conn
				break
			} else {
				// If the connected server is not the leader, continue to the next iteration
				invalidPortsInARow++
				continue
			}

		} else {
			// If not looking for a leader, set 'connection' to the first valid connection and break out of the loop
			connection = conn
			break
		}
	}

	// If no leader is found and 'lookForLeader' is true, recursively call the function with 'lookForLeader' set to false
	if connection == nil && lookForLeader {
		log.Printf("Client %v: No leader found. Retrying without looking for leader\n", client.ID)
		connection = client.findValidServerConn(false)
		return connection
	} else if connection == nil {
		// If no valid connection is found, log and return nil
		log.Printf("Client %v: No valid server found\n", client.ID)
		return nil
	}

	log.Printf("Client %v: Found valid server on port %v\n", client.ID, connection.Target())

	// Return the valid connection, if found
	return connection
}

// Bid submits a bid to the connected server with the specified amount.
// It handles server responses and takes appropriate actions based on the returned acknowledgment.
func (client *AuctionClient) Bid(amount int32) {

	// Perform clientside checks
	if client.HighestBidder == client.ID {
		fmt.Println("You are already the highest bidder")
		return
	} else if amount <= client.HighestBid {
		fmt.Println("Bid is too low")
		return
	}

	log.Printf("Client %v: Attempting to bid %d\n", client.ID, amount)
	// Send a bid request to the server
	ack, err := client.server.Bid(context.Background(), &s.Bid{
		Amount: amount,
		ID:     client.ID,
	})

	// Handle the case where an error occurs during the bid request
	if err != nil {
		// Attempt to reconnect to the server and return
		client.connectToServer(true)
		return
	}

	var response string

	// Handle server acknowledgment
	if !ack.Valid {
		// Process different return codes and take appropriate actions
		switch ack.ReturnCode {
		case 1:
			response = "Another server is the leader. Reconnecting..."
			client.connectToServer(true)
			client.Bid(amount)
			return
		case 2:
			response = "Bid is too low"
			return
		case 3:
			response = "You are already the highest bidder"
			return
		case 4:
			response = "Item is already sold"
			return
		default:
			response = "Unknown return code"
			return
		}
	} else {
		// If the bid is successful, update information
		response = "Bid successful"
		client.Update()
	}

	log.Printf("Client %v: Bid server response: %v\n", client.ID, response)

}

// Update retrieves the latest auction information from the server and updates the client's state.
// It handles server responses and takes appropriate actions based on the returned synchronization data.
func (client *AuctionClient) Update() {
	// Request synchronization data from the server
	sync, err := client.server.Result(context.Background(), &s.Empty{})

	// Handle the case where an error occurs during the synchronization request
	if err != nil {
		// Attempt to reconnect to the server and return
		client.connectToServer(true)
		return
	}

	// Check if the item is sold
	if sync.Sold {
		// Display information about the sold item and terminate the client
		fmt.Printf("Item sold to %s for %d\n", sync.HighestBidder, sync.HighestBid)
		log.Printf("Client: %v: Terminating\n", client.ID)
		os.Exit(0)
	}

	// Check if the highest bid has changed
	if sync.HighestBid != client.HighestBid {
		// Display updated auction information
		fmt.Printf("Highest bid: %d, Highest bidder: %s, Time remaining: %d\n", sync.HighestBid, sync.HighestBidder, sync.TimeRemaining)
	}

	// Update the client's highest bid based on the synchronization data
	client.HighestBid = sync.HighestBid
	client.HighestBidder = sync.HighestBidder
}

// updateIntermittently continuously updates the client's state at regular intervals.
// It calls the Update function and then sleeps for 5 seconds before the next update.
func (client *AuctionClient) updateIntermittently() {
	for {
		// Update the client's state
		client.Update()

		// Sleep for 5 seconds before the next update
		time.Sleep(5 * time.Second)
	}
}

// takeBids continuously prompts the user to enter a bid amount and submits the bid to the server.
// It calls the Bid function with the entered amount.
func (client *AuctionClient) takeBids() {
	for {
		// Prompt the user to enter a bid amount
		var amount int32
		fmt.Println("Please enter the amount you want to bid")

		// Read and validate the user input
		_, err := fmt.Scan(&amount)
		if err != nil {
			fmt.Println("Invalid input")
			continue
		}

		// Submit the bid to the server
		client.Bid(amount)
	}
}
