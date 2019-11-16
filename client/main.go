package main

import (
	"context"
	"fmt"
	"log"

	prompt "github.com/c-bata/go-prompt"
	"github.com/hibooboo2/gchat/api"
	"github.com/hibooboo2/gchat/utils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func main() {
	log.SetFlags(log.Lshortfile)
	// dail server
	conn, err := grpc.Dial(":9090", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("can not connect with server %v", err)
	}
	// create stream
	chatClient := api.NewChatClient(conn)
	authClient := api.NewAuthClient(conn)
	friendsClient := api.NewFriendsClient(conn)

	es := &ExecutorScope{authClient: authClient, chatClient: chatClient, friendClient: friendsClient}
	p := prompt.New(es.executor, Commands)

	p.Run()

	ctx := context.Background()
	// The following code is how you add metadata to the context for the server to use it.
	ctx = metadata.AppendToOutgoingContext(ctx, "TOKEN", "some token from auth")

	statusClient, err := friendsClient.Status(ctx, nil)
	if err != nil {
		panic(err)
	}
	statusClient.Recv()

}

type ExecutorScope struct {
	authClient   api.AuthClient
	ctx          context.Context
	chatClient   api.ChatClient
	friendClient api.FriendsClient
	friendList   map[string]*api.Friend
}

func (e *ExecutorScope) executor(t string) {
	var err error
	fmt.Println("You selected " + t)
	switch t {
	case "register":
		err = reg(e.authClient)
	case "login":
		e.ctx, err = login(e.authClient)

	case "message":
		e.sendMessage()
	case "notifications":
		e.messageNotifications()
	case "send friend request":
		e.sendFriendRequest()
	case "friends list":
		e.getFriends()
	case "remove friend":
		e.removeFriend()
	case "status":
		e.status()
	}
	if err != nil {
		fmt.Println(err)
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
		{Text: "message", Description: "Send a message to a user"},
		{Text: "notifications", Description: "Pull up notifications"},
		{Text: "send friend request", Description: "Send a user a friend request"},
		{Text: "friends list", Description: "Get a list of your friends"},
		{Text: "remove friend", Description: "Removes a friend from your friends list"},
		{Text: "status", Description: "checks the status of friends  from your friends list"},
	}
	return prompt.FilterHasPrefix(s, d.GetWordBeforeCursor(), true)
}

func login(authClient api.AuthClient) (context.Context, error) {

	in := api.LoginRequest{
		Username: prompt.Input("What is your username? ", Empty),
		Password: prompt.Input("What is your password? ", Empty),
	}

	in.Password = utils.Hash(in.Password)

	ctx := context.Background()
	l, err := authClient.Login(ctx, &in)
	if err != nil {
		return nil, err
	}
	ctx = metadata.AppendToOutgoingContext(ctx, "TOKEN", l.Token)

	return ctx, nil
}

func (e *ExecutorScope) sendMessage() {
	_, err := e.chatClient.SendMessage(e.ctx, &api.Message{
		To:   prompt.Input("Who do you want to message?", Empty),
		Data: prompt.Input("What is your message?", Empty),
	})

	if err != nil {
		fmt.Println(err)
		return
	}

}
func (e *ExecutorScope) messageNotifications() {
	stream, err := e.chatClient.Messages(e.ctx, &api.Empty{})
	if err != nil {
		fmt.Println(err)
		return
	}
	go func() {
		msg, err := stream.Recv()
		for err == nil {
			if err == nil {
				fmt.Println(msg.From, msg.Data)
			}
			msg, err = stream.Recv()
		}
	}()
}

func (e *ExecutorScope) sendFriendRequest() {
	_, err := e.friendClient.Add(e.ctx, &api.Friend{
		Username: prompt.Input("Who do you want to send a friend request to?", Empty),
	})
	if err != nil {
		fmt.Println(err)
	}
}

func (e *ExecutorScope) getFriends() {
	e.friendList = map[string]*api.Friend{}
	friends, err := e.friendClient.All(e.ctx, &api.FriendsListReq{})
	if err != nil {
		fmt.Println(err)
	}
	for _, friend := range friends.Friends {
		e.friendList[friend.Username] = friend
	}

}

func (e *ExecutorScope) removeFriend() {
	username := prompt.Input("Which friend do you want to delete?", Empty)
	_, err := e.friendClient.Remove(e.ctx, &api.Friend{
		Username: username,
	})
	if err != nil {
		fmt.Println(err)
		return
	}
	delete(e.friendList, username)

}

func (e *ExecutorScope) status() {
	stream, err := e.friendClient.Status(e.ctx, &api.Empty{})
	if err != nil {
		fmt.Println(err)
	}
	go func() {
		for {
			status, err := stream.Recv()
			if err != nil {
				fmt.Println(err)
				break
			}
			friend, ok := e.friendList[status.Username]
			if ok {
				friend.Status = status.Status
			}
		}
	}()
}
