package auth

import (
	"context"
	"crypto/md5"
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/hibooboo2/gchat/api"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AuthSrv struct {
	users sync.Map
}
type User struct {
	api.RegisterRequest
}

func (a *AuthSrv) ValidToken(token string) (string, bool) {
	vals := strings.SplitN(token, "@", 2)
	if len(vals) != 2 {
		return "", false
	}
	v, ok := a.users.Load(vals[0])
	if !ok {
		return "", false
	}
	u, isUser := v.(User)
	if !isUser {
		return "", false
	}

	if sumString(u.Username+u.Password) != vals[1] {
		return "", false
	}
	return vals[0], true
}

func (a *AuthSrv) Login(ctx context.Context, req *api.LoginRequest) (*api.LoginResponse, error) {
	v, ok := a.users.Load(req.Username)
	u, isUser := v.(User)
	if !ok || !isUser || u.Password != req.Password {
		return nil, status.Errorf(codes.Unauthenticated, "invalid username or password")
	}
	return &api.LoginResponse{Token: req.Username + "@" + sumString(req.Username+req.Password)}, nil
}

func (a *AuthSrv) Register(ctx context.Context, req *api.RegisterRequest) (*api.RegisterResponse, error) {
	_, registered := a.users.Load(req.Username)
	if registered {
		return nil, status.Errorf(codes.AlreadyExists, "username is already registered")
	}
	if len(req.Password) < 2 {
		return nil, status.Error(codes.InvalidArgument, "password is not longer than 2 characters")
	}
	r := regexp.MustCompile(`[0-9A-Za-z_.]`)

	if !r.Match([]byte(req.Username)) {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("username does not match regex: %s", r.String()))
	}

	a.users.Store(req.Username, User{RegisterRequest: *req})
	return &api.RegisterResponse{}, nil
}

func sumString(val string) string {
	h := md5.New()
	h.Write([]byte(val))
	return fmt.Sprintf("%x\n", h.Sum(nil))
}
