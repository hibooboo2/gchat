package main

import (
	"context"
	"log"
	"sync"

	"github.com/hibooboo2/gchat/api"
	"github.com/hibooboo2/gchat/server/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Friends struct {
	db                    *storage.DB
	statusSubscribedUsers sync.Map
}

func (f *Friends) All(ctx context.Context, flr *api.FriendsListReq) (*api.FriendsList, error) {
	return f.db.AllFriends(ctx.Value("USER").(string))
}

func (f *Friends) Add(ctx context.Context, fr *api.Friend) (*api.FriendAddResp, error) {
	if fr.Username == "" {
		return nil, status.Errorf(codes.InvalidArgument, "must provide username in order to add a friend")
	}
	//XXX should return status of friend if they are added and not just a request made.
	err := f.db.AddFriend(ctx.Value("USER").(string), fr.Username)
	if err != nil {
		return nil, err
	}
	return &api.FriendAddResp{}, status.Errorf(codes.OK, "user added")
}

func (f *Friends) Remove(ctx context.Context, friend *api.Friend) (*api.FriendRemoveResp, error) {
	if friend.Username == "" {
		return nil, status.Errorf(codes.InvalidArgument, "must provide username in order to remove a friend")
	}
	return &api.FriendRemoveResp{}, f.db.RemoveFriend(ctx.Value("USER").(string), friend.Username)
}
func (f *Friends) Requests(ctx context.Context, req *api.Empty) (*api.FriendRequests, error) {
	return f.db.GetFriendRequests(ctx.Value("USER").(string))
}
func (f *Friends) Status(m *api.Empty, stream api.Friends_StatusServer) error {
	ctx := stream.Context()
	usr := ctx.Value("USER").(string)

	f.statusSubscribedUsers.Store(usr, stream)
	friends, err := f.db.AllFriends(usr)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to get friends statuses")
	}
	for _, friend := range friends.Friends {
		err := stream.Send(&api.FriendStatus{Username: friend.Username, Status: friend.Status, Online: friend.Online})
		if err != nil {
			return status.Errorf(codes.Internal, "failed to send friend status to user")
		}
	}
	log.Printf("debug: user %s subscribed to status sent %d statuses", usr, len(friends.Friends))
	err = f.db.UpdateStatus(usr, true, "", f.statusSubscribedUsers)
	if err != nil {
		log.Printf("err: %v", err)
	}
	defer func() {
		f.statusSubscribedUsers.Delete(usr)
		err := f.db.UpdateStatus(usr, false, "", f.statusSubscribedUsers)
		if err != nil {
			log.Printf("err: %v", err)
		}
	}()
	<-ctx.Done()
	log.Printf("debug: user %s no longer subscribed to status", usr)

	return nil
}

func (f *Friends) SetStatus(ctx context.Context, su *api.StatusUpdate) (*api.Empty, error) {
	user := ctx.Value("USER").(string)
	err := f.db.UpdateStatus(user, su.Online, su.Status, f.statusSubscribedUsers)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to set status for user %s", user)
	}
	return &api.Empty{}, nil
}
