package main

import (
	"context"
	"fmt"
	"log"

	prompt "github.com/c-bata/go-prompt"
	"github.com/hibooboo2/gchat/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func main() {
	// dail server
	conn, err := grpc.Dial(":9090", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("can not connect with server %v", err)
	}
	// create stream
	chatClient := api.NewChatClient(conn)
	authClient := api.NewAuthClient(conn)
	friendsClient := api.NewFriendsClient(conn)

	es := &ExecutorScope{authClient: authClient}
	p := prompt.New(es.executor, Commands)

	p.Run()

	ctx := context.Background()
	// The following code is how you add metadata to the context for the server to use it.
	ctx = metadata.AppendToOutgoingContext(ctx, "TOKEN", "some token from auth")
	resp, err := chatClient.SendMessage(ctx, &api.Message{Data: "some string"})

	if err != nil {
		log.Fatalln(err)
	}

	log.Println("response was:", resp)

	statusClient, err := friendsClient.Status(ctx, nil)
	if err != nil {
		panic(err)
	}
	statusClient.Recv()

}

type ExecutorScope struct {
	authClient api.AuthClient
	ctx        context.Context
}

func (e *ExecutorScope) executor(t string) {
	var err error
	fmt.Println("You selected " + t)
	switch t {
	case "register":
		err = reg(e.authClient)
	case "login":
		e.ctx, err = login(e.authClient)
	}
	if err != nil {
		log.Fatal(err)
	}
}

func reg(authClient api.AuthClient) error {
	ctx := context.Background()
	req := &api.RegisterRequest{}
	req.Email = prompt.Input("What is your email?", Empty)
	req.Username = prompt.Input("What is your desired username?", Empty)
	req.Password = prompt.Input("What do you want to set your password as?", Empty)
	req.FirstName = prompt.Input("(Optional)What is your first name ?", Empty)
	req.LastName = prompt.Input("(Optional)What is your last name?", Empty)
	regResp, err := authClient.Register(ctx, req)
	if err != nil {
		return err
	}
	log.Println(regResp)
	return nil
}

func Empty(d prompt.Document) []prompt.Suggest { return nil }

func Commands(d prompt.Document) []prompt.Suggest {
	s := []prompt.Suggest{
		{Text: "register", Description: "Register a user"},
		{Text: "login", Description: "Login a user"},
	}
	return prompt.FilterHasPrefix(s, d.GetWordBeforeCursor(), true)
}

func login(authClient api.AuthClient) (context.Context, error) {

	in := api.LoginRequest{
		Username: prompt.Input("What is your username? ", Empty),
		Password: prompt.Input("What is your password? ", Empty),
	}
	ctx := context.Background()
	l, err := authClient.Login(ctx, &in)
	if err != nil {
		return nil, err
	}
	ctx = metadata.AppendToOutgoingContext(ctx, "TOKEN", l.Token)

	return ctx, nil
}
