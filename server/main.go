package main

import (
	"context"
	"log"
	"net"

	"github.com/hibooboo2/gchat/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func main() {
	lis, err := net.Listen("tcp", ":9090")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	defer lis.Close()

	s := grpc.NewServer()
	api.RegisterChatServer(s, &Server{})

	// and start...
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

type Server struct {
}

func (s *Server) SendMessage(ctx context.Context, m *api.Message) (*api.MessageResp, error) {
	log.Println("data:", m.Data)
	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		log.Println(md.Get("TOKEN")[0])
	}
	return &api.MessageResp{Data: m.Data}, nil
}
