package main

import (
	"log"
	"net"
	"sync"

	"github.com/comail/colog"
	"github.com/hibooboo2/gchat/api"
	"github.com/hibooboo2/gchat/server/auth"
	"github.com/hibooboo2/gchat/server/storage"
	"google.golang.org/grpc"
)

func main() {
	log.SetFlags(log.Lshortfile)
	colog.Register()
	colog.SetDefaultLevel(colog.LDebug)
	colog.SetMinLevel(colog.LTrace)
	lis, err := net.Listen("tcp", ":9090")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	defer lis.Close()

	db, err := storage.New("gchat.db")
	if err != nil {
		log.Fatalln(err)
	}
	statusSubscribedUsers := sync.Map{}
	messageSubscribedUsers := sync.Map{}
	a := auth.New(db)

	chat := NewChatServer(db, a.ValidToken, messageSubscribedUsers)

	s := grpc.NewServer(grpc.UnaryInterceptor(chat.ServerAuthInterceptor), grpc.StreamInterceptor(chat.ServerStreamAuthInterceptor))
	api.RegisterChatServer(s, chat)
	api.RegisterAuthServer(s, a)
	api.RegisterFriendsServer(s, &Friends{db, statusSubscribedUsers})

	// and start...
	if err := s.Serve(lis); err != nil {
		log.Fatalf("err: failed to serve: %v", err)
	}
}
