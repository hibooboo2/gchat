package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"runtime/debug"
	"time"

	"github.com/gdamore/tcell"
	"github.com/hibooboo2/gchat/api"
	"github.com/hibooboo2/gchat/utils"
	"github.com/rivo/tview"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

var alreadyRanMain bool

func main() {
	defer func() {
		err := recover()
		if err != nil {
			f, _ := os.Create("died.txt")
			f.WriteString(fmt.Sprintf("%#+v %v", err, string(debug.Stack())))
			f.Close()
		}
	}()
	if alreadyRanMain {
		fmt.Println("Main already ran")
	}
	log.SetFlags(log.Lshortfile | log.Ltime | log.Ldate)
	// dail server
	conn, err := grpc.Dial(":9090", grpc.WithInsecure(), grpc.WithTimeout(time.Millisecond))
	if err != nil {
		log.Fatalf("can not connect with server %v", err)
	}
	defer conn.Close()
	// create stream
	chatClient := api.NewChatClient(conn)
	authClient := api.NewAuthClient(conn)
	friendsClient := api.NewFriendsClient(conn)

	cc := &ChatClient{authClient: authClient, chatClient: chatClient, friendClient: friendsClient}
	cc.app = tview.NewApplication()

	cc.logOut()
	if err := cc.app.Run(); err != nil {
		panic(err)
	}
}

func (cc *ChatClient) logOut() {
	cc.users = &Users{list: map[string]*UserInfo{}}

	cc.grid = tview.NewGrid()
	tv := tview.NewTextView()
	log.SetOutput(tv)
	tv.SetBorder(true)
	tv.SetScrollable(true)
	tv.SetChangedFunc(func() {
		tv.ScrollToEnd()
	})
	cc.grid.AddItem(tv, 0, 0, 1, 3, 0, 0, false)

	cc.notifications = tview.NewTextView()
	cc.notifications.SetTitle("Notifications")
	cc.notifications.SetBorder(true)
	cc.notifications.SetBorderColor(tcell.ColorGreen)
	cc.notifications.SetScrollable(true)
	cc.notifications.SetChangedFunc(func() {
		cc.notifications.ScrollToEnd()
	})

	cc.grid.AddItem(cc.notifications, 0, 3, 2, 1, 0, 0, false)

	cc.userChats = tview.NewPages()
	cc.userChats.SetBorder(true)
	cc.grid.AddItem(cc.userChats, 1, 1, 3, 2, 0, 0, false)

	list := tview.NewList()
	list.AddItem("login", "Login as a user", 'l', cc.loginHandler)
	list.AddItem("register", "Register a new user", 'r', cc.reg)
	cc.useNewKeys(list)

	cc.app.SetRoot(cc.hotkeys, true).SetFocus(cc.hotkeys).Draw()
}

func (cc *ChatClient) useNewKeys(list *tview.List) {
	if cc.hotkeys != nil {
		cc.grid.RemoveItem(cc.hotkeys)
	}
	cc.commonHotkeys(list)
	cc.hotkeys = list
	cc.grid.AddItem(cc.hotkeys, 2, 3, 2, 1, 0, 0, true)
}

func (cc *ChatClient) commonHotkeys(list *tview.List) {
	list.AddItem("Help", "Press to bring up this screen", 'h', func() {
		cc.app.SetRoot(cc.grid, true).SetFocus(list).Draw()
	})
	list.AddItem("Quit", "Press to exit", 'q', func() {
		cc.app.Stop()
	})
}

func (cc *ChatClient) addFriendChat(username string) {
	if !cc.userChats.HasPage(username) {
		g := tview.NewGrid()
		tv := tview.NewTextView()
		tv.SetTitle(username)
		tv.SetBorder(true)
		tv.SetChangedFunc(func() {
			tv.ScrollToEnd()
		})
		cc.users.Get(username).tv = tv

		n := tview.NewTreeNode(username).SetSelectable(true)
		cc.users.Get(username).node = n
		cc.friendsNode.AddChild(n)
		n.SetReference(func() {
			cc.users.Get(username).Update(true)
			cc.userChats.SwitchToPage(username)
			cc.app.SetRoot(cc.grid, true).SetFocus(cc.userChats).Draw()
		})

		msgs, err := cc.chatClient.MessagesWith(cc.ctx, &api.Friend{Username: username})
		if err == nil {
			for _, msg := range msgs.Messages {
				fmt.Fprintf(tv, "%s: %s\n", msg.From, msg.Data)
			}
		}

		g.AddItem(tv, 0, 0, 9, 1, 0, 0, false)
		in := tview.NewInputField()
		msg := ""
		in.SetChangedFunc(func(text string) {
			msg = text
		})
		in.SetDoneFunc(func(key tcell.Key) {
			switch key {
			// case tcell.KeyTab:
			case tcell.KeyEscape:
				cc.app.SetFocus(cc.userTree)
			case tcell.KeyEnter:
				cc.sendMessage(username, msg)
				fmt.Fprintf(tv, "%s: %s\n", cc.user, msg)
				in.SetText("")
			}
		})
		g.AddItem(in, 9, 0, 1, 1, 0, 0, true)
		cc.userChats.AddPage(username, g, true, false)
		log.Println("Added:", username)
	}
}

func (cc *ChatClient) loginHandler() {
	form := tview.NewForm()
	username := ""
	password := ""
	form.AddInputField("Username:", "", 100, tview.InputFieldMaxLength(36), func(text string) {
		username = text
	})
	form.AddPasswordField("Password:", "", 0, '*', func(text string) {
		password = text
	})
	form.AddButton("login", func() {
		list := tview.NewList()
		err := cc.login(username, password)
		if err != nil {
			m := tview.NewModal().SetText(err.Error())
			m.AddButtons([]string{"OK"}).SetDoneFunc(func(buttonIndex int, buttonLabel string) {
				cc.app.SetRoot(cc.hotkeys, true).SetFocus(cc.hotkeys).Draw()
			})
			cc.app.SetRoot(m, false).SetFocus(m).Draw()
			return
		}
		list.AddItem("logout", "Logout from "+username, 'l', cc.logOut)
		list.AddItem("add friend", "Send a friend request to a user", 'a', func() {
			form := tview.NewForm()
			userToAdd := ""
			form.AddInputField("Username", "", 0, tview.InputFieldMaxLength(36), func(text string) {
				userToAdd = text
			})
			form.AddButton("Send friend request", func() {
				log.Println("Sent friend request")
				cc.sendFriendRequest(userToAdd)
				cc.app.SetRoot(cc.grid, true).SetFocus(cc.hotkeys)
			})
			form.SetInputCapture(func(key *tcell.EventKey) *tcell.EventKey {
				switch key.Key() {
				case tcell.KeyEsc:
					cc.app.SetRoot(cc.grid, true).SetFocus(cc.hotkeys)
					return nil
				}
				return key
			})
			cc.app.SetRoot(form, true).SetFocus(form).Draw()
		})
		list.AddItem("status", "Set your online status", 's', func() {
			form := tview.NewForm()
			status := ""
			form.AddInputField("Status", "", 0, tview.InputFieldMaxLength(36), func(text string) {
				status = text
			})
			online := true
			form.AddCheckbox("Show As Online", true, func(checked bool) {
				online = checked
			})
			form.AddButton("Set Status", func() {
				cc.updateStatus(status, online)
				cc.app.SetRoot(cc.grid, true).SetFocus(cc.hotkeys)
			})
			form.AddButton("Cancel", func() {
				cc.app.SetRoot(cc.grid, true).SetFocus(cc.hotkeys)
			})
			cc.app.SetRoot(form, true).SetFocus(form).Draw()
		})

		cc.userTree = tview.NewTreeView()
		cc.userTree.SetBorder(true)
		cc.grid.AddItem(cc.userTree, 1, 0, 3, 1, 0, 0, false)
		list.AddItem("message", "Message a user of your choice", 'm', func() {
			cc.app.SetRoot(cc.grid, true).SetFocus(cc.userTree).Draw()
		})

		base := tview.NewTreeNode("-")
		cc.friendsNode = tview.NewTreeNode("Friends")
		requests := tview.NewTreeNode("Requests")
		base.AddChild(requests)
		base.AddChild(cc.friendsNode)

		for _, f := range cc.friends {
			cc.addFriendChat(f.Username)
		}
		cc.userTree.SetRoot(base).SetCurrentNode(base)
		cc.userTree.SetSelectedFunc(func(node *tview.TreeNode) {
			log.Println("Tree Selection:", node.GetText())
			focus, ok := node.GetReference().(func())
			if !ok {
				log.Println("Selected node does not have a callback")
				return
			}
			focus()
		})
		cc.userTree.SetInputCapture(func(key *tcell.EventKey) *tcell.EventKey {
			switch key.Key() {
			case tcell.KeyEsc:
				cc.app.SetFocus(list)
			}
			return key
		})

		cc.getStatus()
		cc.messageNotifications(cc.userTree)
		cc.getFriendRequests(requests)
		list.AddItem("Check Friend Requests", "Check to see if you have any new friend requests", 'r', func() {
			cc.getFriendRequests(requests)
		})

		cc.useNewKeys(list)
		cc.app.SetRoot(cc.grid, true).SetFocus(cc.userTree).Draw()
	})

	cc.app.SetRoot(form, true).SetFocus(form).Draw()
}

type ChatClient struct {
	app           *tview.Application
	authClient    api.AuthClient
	chatClient    api.ChatClient
	ctx           context.Context
	friendClient  api.FriendsClient
	friends       []*api.Friend
	friendsNode   *tview.TreeNode
	grid          *tview.Grid
	hotkeys       *tview.List
	notifications *tview.TextView
	user          string
	userChats     *tview.Pages
	users         *Users
	userTree      *tview.TreeView
}
type Users struct {
	list map[string]*UserInfo
}

func (u *Users) Get(username string) *UserInfo {
	user, ok := u.list[username]
	if !ok {
		user = &UserInfo{username: username}
		u.list[username] = user
		log.Println("returned and empty user info")
	}
	return user
}

type UserInfo struct {
	tv             *tview.TextView
	node           *tview.TreeNode
	status         string
	online         bool
	unreadMessages int
	username       string
}

func (u *UserInfo) Focused() {
	u.Update(true)
}

func (u *UserInfo) UpdateStatus(status string, online bool, isFocused bool) {
	u.status = status
	u.online = online
	u.Update(isFocused)
}

func (u *UserInfo) Update(isFocused bool) {
	if isFocused {
		u.unreadMessages = 0
	}

	node := u.node
	display := u.username

	switch u.online {
	case true:
		display = fmt.Sprintf("❇️ %s", display)
		// node.SetColor(tcell.ColorGreen)
	case false:
		display = fmt.Sprintf("🔴 %s", display)
		// node.SetColor(tcell.ColorRed)
	}

	if !isFocused && u.unreadMessages > 0 {
		display = fmt.Sprintf("%s(%d)", display, u.unreadMessages)
	}
	switch u.status {
	case "", "offline", "online":
	default:
		display = fmt.Sprintf("%s: %s", display, u.status)
	}
	node.SetText(display)
}

func (u *UserInfo) AddUnread(unreadMessages int, isFocused bool) {

	u.unreadMessages += unreadMessages
	u.Update(isFocused)
}

func (cc *ChatClient) reg() {
	form := tview.NewForm()
	req := &api.RegisterRequest{}

	form.AddInputField("(Optional) What is your first name ?", "", 0, tview.InputFieldMaxLength(50), func(first string) {
		req.FirstName = first
	})
	form.AddInputField("(Optional) What is your last name?", "", 0, tview.InputFieldMaxLength(50), func(last string) {
		req.LastName = last
	})
	form.AddInputField("What is your email?", "", 0, tview.InputFieldMaxLength(200), func(email string) {
		req.Email = email
	})
	form.AddInputField("What is your desired username?", "", 0, tview.InputFieldMaxLength(36), func(username string) {
		req.Username = username
	})
	form.AddPasswordField("What do you want to set your password as?", "", 0, '*', func(password string) {
		req.Password = password
	})
	prev := cc.app.GetFocus()
	cc.app.SetRoot(form, true).SetFocus(form).Draw()
	form.AddButton("Register", func() {
		ctx := context.Background()
		regResp, err := cc.authClient.Register(ctx, req)
		if err != nil {
			log.Println(err)
			cc.app.SetRoot(prev, true).SetFocus(prev).Draw()
			return
		}
		log.Println(regResp)
		cc.app.SetRoot(prev, true).SetFocus(prev).Draw()
	})
}

func (cc *ChatClient) login(username string, password string) error {
	in := api.LoginRequest{
		Username: username,
		Password: password,
	}

	in.Password = utils.Hash(in.Password)

	ctx := context.Background()
	l, err := cc.authClient.Login(ctx, &in)
	if err != nil {
		return err
	}
	cc.ctx = metadata.AppendToOutgoingContext(ctx, "TOKEN", l.Token)
	cc.user = username
	cc.getFriends()
	return nil
}

func (cc *ChatClient) sendMessage(user string, message string) {
	_, err := cc.chatClient.SendMessage(cc.ctx, &api.Message{
		To:   user,
		Data: message,
	})
	if err != nil {
		log.Println(err)
		return
	}
}

func (cc *ChatClient) messageNotifications(tree *tview.TreeView) {
	stream, err := cc.chatClient.Messages(cc.ctx, &api.Empty{})
	if err != nil {
		log.Println(err)
		return
	}
	go func() {
		msg, err := stream.Recv()
		for err == nil {
			if msg.From != cc.user {
				cc.addFriendChat(msg.From)
				u := cc.users.Get(msg.From)
				fmt.Fprintf(u.tv, "%s: %s\n", msg.From, msg.Data)
				u.AddUnread(1, tree.GetCurrentNode() == u.node)
				fmt.Fprintf(cc.notifications, "Message from: %s\n", msg.From)
				cc.app.Draw()
			}
			msg, err = stream.Recv()
		}
	}()
}

func (cc *ChatClient) sendFriendRequest(username string) {
	_, err := cc.friendClient.Add(cc.ctx, &api.Friend{
		Username: username,
	})
	if err != nil {
		log.Println(err)
	}
}

func (cc *ChatClient) getFriends() {
	friends, err := cc.friendClient.All(cc.ctx, &api.FriendsListReq{})
	if err != nil {
		log.Println(err)
		return
	}
	cc.friends = friends.Friends
}

func (cc *ChatClient) getFriendRequests(node *tview.TreeNode) {
	node.ClearChildren()
	requests, err := cc.friendClient.Requests(cc.ctx, &api.Empty{})
	if err != nil {
		log.Println("err: ", err)
		return
	}
	log.Println("Got requests", len(requests.Friends))
	for _, req := range requests.Friends {
		r := *req
		n := tview.NewTreeNode(r.Username).SetColor(tcell.ColorLightCyan)
		node.AddChild(n)
		n.SetReference(func() {
			prev := cc.app.GetFocus()
			m := tview.NewModal().AddButtons([]string{"Add friend", "Cancel"})
			m.SetBorder(true)
			m.SetText("Would you like to add " + r.Username)
			m.SetDoneFunc(func(choice int, label string) {
				if choice == 0 {
					_, err := cc.friendClient.Add(cc.ctx, &r)
					if err != nil {
						log.Println(err)
					}
					cc.addFriendChat(r.Username)
					cc.getFriendRequests(node)
				}
				cc.app.SetRoot(cc.grid, true).SetFocus(prev).Draw()
			})
			cc.app.SetRoot(m, true).SetFocus(m).Draw()
		})
		log.Println("Added friend request to nodes", r.Username)
	}

}

func (cc *ChatClient) getStatus() {
	stream, err := cc.friendClient.Status(cc.ctx, &api.Empty{})
	if err != nil {
		log.Println(err)
		return
	}
	go func() {
		for status, err := stream.Recv(); err == nil; status, err = stream.Recv() {
			node := cc.users.Get(status.Username).node
			if node == nil {
				log.Println("No node found for user", status.Username)
				continue
			}
			u := cc.users.Get(status.Username)

			if u.status != status.Status || u.online != status.Online {
				u.UpdateStatus(status.Status, status.Online, cc.userTree.GetCurrentNode() == node)
				if status.Online {
					fmt.Fprintf(cc.notifications, "%s is now online\n", status.Username)
				} else {
					fmt.Fprintf(cc.notifications, "%s is now offline\n", status.Username)
				}

			}
			cc.app.Draw()

		}
	}()
}

func (cc *ChatClient) updateStatus(status string, online bool) {
	_, err := cc.friendClient.SetStatus(cc.ctx, &api.StatusUpdate{Status: status, Online: online})
	if err != nil {
		log.Println(err)
		return
	}
}
