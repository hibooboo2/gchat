package main

import (
	"context"
	"errors"
	"log"

	"github.com/hibooboo2/gchat/api"
	"github.com/hibooboo2/gchat/server/storage"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type Server struct {
	validateToken func(string) (string, bool)
	db            *storage.DB
}

func NewChatServer(db *storage.DB, validateToken func(string) (string, bool)) *Server {
	return &Server{db: db, validateToken: validateToken}
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
	user, ok := s.validateToken(t)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "invalid authentication token")
	}
	ctx = context.WithValue(ctx, "USER", user)

	resp, err := handler(ctx, methodreq)
	if err != nil {
		log.Printf("%#+v", err)
	}
	return resp, err
}

func (s *Server) SendMessage(ctx context.Context, m *api.Message) (*api.MessageResp, error) {
	user := ctx.Value("USER").(string)
	return &api.MessageResp{Data: m.Data}, s.db.SaveMessage(m, user)
}

func (s *Server) MessagesWith(ctx context.Context, f *api.Friend) (*api.MessageList, error) {
	user := ctx.Value("USER").(string)
	return s.db.GetMessages(user, f.Username)
}
func (s *Server) SendMessageToRoom(context.Context, *api.RoomMessage) (*api.MessageResp, error) {
	return nil, errors.New("not implemented")

}
func (s *Server) RoomMessages(context.Context, *api.Room) (*api.MessageList, error) {
	return nil, errors.New("not implemented")

}
func (s *Server) Messages(*api.Empty, api.Chat_MessagesServer) error {
	return errors.New("not implemented")

}
