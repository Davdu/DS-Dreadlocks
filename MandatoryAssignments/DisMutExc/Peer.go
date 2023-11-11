package main

import (
	"chittychat/chittychat"
	"log"
)

type node struct {
	ID          int64
	LamportTime int64
	peers []proto.
}

func main() {

}

func Join() {

}

func RequestToken() {

}

func CallElection() {

}


// Compares and returns the largest int32.
func max(a, b int32) int32 {
	if a > b {
		return a
	}
	return b
}

func (s *node) Messages(stream proto.Proto_) error {
	s.clients = append(s.clients, stream)

	// infinite loop
	for {
		//Constantly recieves messages on streams and broadcasts them immediately
		// If a stream is met with an exception, Clients are removed
		incoming, err := stream.Recv()
		if err != nil {
			s.removeClientAndNotify(stream)
			return err
		}

		incoming.Timestamp = s.updateAndIncrementLamport(incoming.Timestamp)
		incoming.Message = "Client "
		log.Printf("Server, Incoming  -  %s", incoming)
		s.usernames[stream] = incoming.Username
		s.broadcastMessage(incoming)
	}
}