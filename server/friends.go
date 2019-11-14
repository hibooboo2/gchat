package main

import (
	"context"

	"github.com/hibooboo2/gchat/api"
	"github.com/hibooboo2/gchat/server/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Friends struct {
	db *storage.DB
}

func (f *Friends) All(ctx context.Context, flr *api.FriendsListReq) (*api.FriendsList, error) {
	return f.db.AllFriends(ctx.Value("USER").(string))
}

func (f *Friends) Add(ctx context.Context, fr *api.Friend) (*api.FriendAddResp, error) {
	if fr.Username == "" {
		return nil, status.Errorf(codes.InvalidArgument, "must provide username in order to add a friend")
	}
	//XXX should return status of friend if they are added and not just a request made.
	return &api.FriendAddResp{}, f.db.AddFriend(ctx.Value("USER").(string), fr.Username)
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
func (f *Friends) Status(*api.Empty, api.Friends_StatusServer) error {
	return nil
}
