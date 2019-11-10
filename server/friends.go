package main

import (
	"context"

	"github.com/hibooboo2/gchat/api"
)

type Friends struct {
}

func (f *Friends) All(context.Context, *api.FriendsListReq) (*api.FriendsList, error) {
	return nil, nil
}
func (f *Friends) Add(context.Context, *api.Friend) (*api.FriendAddResp, error) {
	return nil, nil
}
func (f *Friends) Remove(context.Context, *api.Friend) (*api.FriendRemoveResp, error) {
	return nil, nil
}
func (f *Friends) Requests(context.Context, *api.Empty) (*api.FriendRequests, error) {
	return nil, nil
}
func (f *Friends) Status(*api.Empty, api.Friends_StatusServer) error {
	return nil
}
