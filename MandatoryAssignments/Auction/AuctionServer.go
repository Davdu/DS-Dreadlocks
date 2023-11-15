package main

import (
	s "Auction/Server"
	"log"
	"os"
)

type AuctionServer struct {
	s.UnimplementedAuctionServer
	highestBid    int
	highestBidder int
	isLeader      bool
	backupServers []s.AuctionServer
}

func main() {

	// Create a logger, Courtesy of https://stackoverflow.com/questions/19965795/how-to-write-log-to-file
	f, err := os.OpenFile("LogFile", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0664)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer f.Close()
	log.SetOutput(f)

}
