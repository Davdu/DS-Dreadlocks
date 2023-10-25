package main

import (
	"chittychat/chittychat"
	"fmt"
	"google.golang.org/grpc"
	"log"
	"net"
	"os"
)

type server struct {
	chittychat.UnimplementedChittyChatServer
	clients      []chittychat.ChittyChat_MessagesServer
	usernames    map[chittychat.ChittyChat_MessagesServer]string
	lamportClock int32
}

func (s *server) updateAndIncrementLamport(receivedTimestamp int32) int32 {
	s.lamportClock = max(s.lamportClock, receivedTimestamp) + 1
	return s.lamportClock
}

func max(a, b int32) int32 {
	if a > b {
		return a
	}
	return b
}

func (s *server) Messages(stream chittychat.ChittyChat_MessagesServer) error {
	s.clients = append(s.clients, stream)

	// infinite loop
	for {
		incoming, err := stream.Recv()
		if err != nil {
			s.removeClientAndNotify(stream)
			return err
		}

		incoming.Timestamp = s.updateAndIncrementLamport(incoming.Timestamp)
		s.usernames[stream] = incoming.Username

		s.broadcastMessage(incoming)
	}
}

func (s *server) removeClientAndNotify(targetStream chittychat.ChittyChat_MessagesServer) {

	s.updateAndIncrementLamport(s.lamportClock)

	for i, client := range s.clients {
		if client == targetStream {
			s.clients = append(s.clients[:i], s.clients[i+1:]...)

			username := s.usernames[client]
			disconnectMsg := &chittychat.Message{
				Username:  "Server",
				Message:   fmt.Sprintf("%s left Chitty-Chat", username),
				Timestamp: s.lamportClock,
			}

			log.Printf("Server  -  %s", disconnectMsg)

			s.broadcastMessage(disconnectMsg)
			break
		}
	}
}

func (s *server) broadcastMessage(msg *chittychat.Message) {

	log.Printf("Server, Outgoing  -  %s", msg)

	for _, client := range s.clients {
		if err := client.Send(msg); err != nil {
			log.Printf("Failed to broadcast message: %v", err)
		}
	}
}

func main() {

	// Create a logger, Courtesy of https://stackoverflow.com/questions/19965795/how-to-write-log-to-file
	f, err := os.OpenFile("LogFile", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0664)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer f.Close()
	log.SetOutput(f)

	lis, _ := net.Listen("tcp", ":50051")
	grpcServer := grpc.NewServer()

	service := &server{
		usernames: make(map[chittychat.ChittyChat_MessagesServer]string),
	}

	chittychat.RegisterChittyChatServer(grpcServer, service)
	grpcServer.Serve(lis)
}
