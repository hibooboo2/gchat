package main

import (
	"context"
	"errors"
	"log"
	"strings"
	"sync"

	"github.com/hibooboo2/gchat/api"
	"github.com/hibooboo2/gchat/server/storage"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type Server struct {
	validateToken          func(string) (string, bool)
	db                     *storage.DB
	messageSubscribedUsers sync.Map
}

func NewChatServer(db *storage.DB, validateToken func(string) (string, bool), messageSubscribedUsers sync.Map) *Server {
	return &Server{db: db, validateToken: validateToken, messageSubscribedUsers: messageSubscribedUsers}
}

func (s *Server) AuthApplier(ctx context.Context, method string) (context.Context, error) {
	if strings.HasPrefix(method, "/api.Auth/") {
		return ctx, nil
	}
	log.Println("trace: api called: ", method)

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "missing authentication token")
	}

	t := md.Get("TOKEN")
	if len(t) != 1 {
		return nil, status.Errorf(codes.Unauthenticated, "missing authentication token in metadata")
	}
	user, ok := s.validateToken(t[0])
	if !ok || user == "" {
		return nil, status.Errorf(codes.Unauthenticated, "invalid authentication token")
	}

	ctx = context.WithValue(ctx, "USER", user)
	return ctx, nil
}

type StreamWithContext struct {
	grpc.ServerStream
	ctx context.Context
}

func (s StreamWithContext) Context() context.Context {
	return s.ctx
}

func (s *Server) ServerStreamAuthInterceptor(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	ctx, err := s.AuthApplier(ss.Context(), info.FullMethod)
	if err != nil {
		return err
	}
	ss = &StreamWithContext{ss, ctx}
	err = handler(srv, ss)
	if err != nil {
		log.Printf("%#+v", err)
	}
	return err
}

func (s *Server) ServerAuthInterceptor(ctx context.Context, methodreq interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	var err error
	ctx, err = s.AuthApplier(ctx, info.FullMethod)
	if err != nil {
		return nil, err
	}
	resp, err := handler(ctx, methodreq)
	if err != nil {
		log.Printf("%#+v", err)
	}
	return resp, err
}

func (s *Server) SendMessage(ctx context.Context, m *api.Message) (*api.MessageResp, error) {
	user := ctx.Value("USER").(string)
	log.Println("trace: Received request to send message")
	m.From = user
	conn, ok := s.messageSubscribedUsers.Load(m.To)
	if ok {
		log.Printf("trace: found connection for user. Sending message to %s", m.To)
		err := conn.(api.Chat_MessagesServer).Send(m)
		if err != nil {
			log.Println("err: failed to send message", err)
		}
	} else {
		log.Printf("trace: failed to find connection for user just storing message for: %s", m.To)
	}
	return &api.MessageResp{Data: m.Data}, s.db.SaveMessage(m, user)
}

func (s *Server) MessagesWith(ctx context.Context, f *api.Friend) (*api.MessageList, error) {
	user := ctx.Value("USER").(string)
	return s.db.GetMessages(user, f.Username)
}

func (s *Server) SendMessageToRoom(context.Context, *api.Message) (*api.MessageResp, error) {
	return nil, errors.New("not implemented")

}

func (s *Server) RoomMessages(context.Context, *api.Room) (*api.MessageList, error) {
	return nil, errors.New("not implemented")

}

func (s *Server) Messages(r *api.Empty, stream api.Chat_MessagesServer) error {
	ctx := stream.Context()
	usr, ok := ctx.Value("USER").(string)
	if !ok {
		log.Println("err: still not getting user")
		return nil
	}
	s.messageSubscribedUsers.Store(usr, stream)
	// stream.Send(&api.Message{Data: "You are now subscribed to notifications for messages"})
	<-ctx.Done()
	s.messageSubscribedUsers.Delete(usr)
	log.Printf("trace: subscribed user: %s is done", usr)
	return nil

}
