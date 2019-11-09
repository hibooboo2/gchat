package main

import (
	"context"
	"log"

	"github.com/hibooboo2/gchat/api"
	"google.golang.org/grpc"
)

func main() {
	// dail server
	conn, err := grpc.Dial(":9090", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("can not connect with server %v", err)
	}

	// create stream
	chatClient := api.NewChatClient(conn)

	ctx := context.Background()
	resp, err := chatClient.SendMessage(ctx, &api.Message{Data: "some string"})

	if err != nil {
		log.Fatalln(err)
	}
	log.Println("response was:", resp)
}
