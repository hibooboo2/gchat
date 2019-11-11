package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	prompt "github.com/c-bata/go-prompt"
	"github.com/fatih/color"
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
	es := &ExecutorScope{}
	es.chatClient = api.NewChatClient(conn)
	es.authClient = api.NewAuthClient(conn)
	es.friendsClient = api.NewFriendsClient(conn)

	p := prompt.New(es.executor, Commands)

	p.Run()
}

type ExecutorScope struct {
	authClient    api.AuthClient
	chatClient    api.ChatClient
	friendsClient api.FriendsClient
	ctx           context.Context
}

func (e *ExecutorScope) executor(t string) {
	var err error
	switch t {
	case "message", "messages":
		if e.ctx == nil {
			log.Println("You are not logged in please login first")
			return
		}
	}
	vals := strings.Split(t, " ")
	cmd := vals[0]
	switch cmd {
	case "register":
		err = reg(e.authClient)
	case "login":
		e.ctx, err = login(e.authClient)
	case "message":
		err = message(e.chatClient, e.ctx, vals[1:])
	case "messages":
		err = getMessages(e.chatClient, e.ctx)
	case "msgs":
		err = startMsgNotifications(e.chatClient, e.ctx)
	default:
		fmt.Println("You selected " + t + " which is an invalid command")
	}
	if err != nil {
		log.Printf("%+#v", err)
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
		{Text: "message", Description: "Message a user"},
		{Text: "messages", Description: "Get all messsages from user"},
		{Text: "msgs", Description: "Subscribe to message notifications for messages sent to currently logged  in user"},
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
	ctx = context.WithValue(ctx, "username", in.Username)
	log.Println(l.Token)
	return ctx, nil
}

func message(cc api.ChatClient, ctx context.Context, args []string) error {
	var err error
	if len(args) < 2 {
		_, err = cc.SendMessage(ctx, &api.Message{
			Data: prompt.Input("What is your message? ", Empty),
			To:   prompt.Input("Who do you want to send it to? ", Empty),
		})
	} else {
		_, err = cc.SendMessage(ctx, &api.Message{Data: strings.Join(args[1:], " "), To: args[0]})
	}
	return err
}

func startMsgNotifications(cc api.ChatClient, ctx context.Context) error {
	msgs, err := cc.Messages(ctx, &api.Empty{})
	if err != nil {
		return err
	}
	go func() {
		for {
			msg, err := msgs.Recv()
			if err != nil {
				log.Println("Errored receiving message", err)
				break
			}
			printMessage(ctx, msg)
		}
	}()
	return nil
}
func getMessages(cc api.ChatClient, ctx context.Context) error {
	messages, err := cc.MessagesWith(ctx, &api.Friend{
		Username: prompt.Input("Who check messages from?", Empty),
	})
	if err != nil {
		return err
	}
	for _, m := range messages.Messages {
		printMessage(ctx, m)
	}
	return nil
}

func printMessage(ctx context.Context, m *api.Message) {
	green := color.New(color.FgYellow).SprintFunc()
	blue := color.New(color.FgBlue).SprintFunc()

	currentUser := ctx.Value("username").(string)
	if m.To == currentUser {
		log.Printf("%s: %s", green(m.From), m.Data)
	} else {
		log.Printf("%s: %s", blue(m.From), m.Data)
	}
}
