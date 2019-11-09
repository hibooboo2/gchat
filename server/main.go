package main

import (
	"context"
	"log"
	"net"

	"github.com/hibooboo2/gchat/api"
	"github.com/hibooboo2/gchat/server/auth"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func main() {
	lis, err := net.Listen("tcp", ":9090")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	defer lis.Close()

	a := auth.AuthSrv{}
	chat := Server{Auth: &a}

	s := grpc.NewServer(grpc.UnaryInterceptor(chat.ServerAuthInterceptor))
	api.RegisterChatServer(s, &chat)
	api.RegisterAuthServer(s, &a)

	// and start...
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

type Server struct {
	Auth *auth.AuthSrv
}

func (s *Server) SendMessage(ctx context.Context, m *api.Message) (*api.MessageResp, error) {
	user := ctx.Value("USER").(string)
	log.Println(user, "sent message", m.Data)
	return &api.MessageResp{Data: m.Data}, nil
}

func (s *Server) ServerAuthInterceptor(ctx context.Context, methodreq interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	log.Println(info.Server, info.FullMethod)
	if info.Server != s {
		return handler(ctx, methodreq)
	}
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "missing authentication token")
	}

	t := md.Get("TOKEN")[0]
	user, ok := s.Auth.ValidToken(t)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "invalid authentication token")
	}
	ctx = context.WithValue(ctx, "USER", user)

	return handler(ctx, methodreq)
}
