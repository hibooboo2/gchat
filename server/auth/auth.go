package auth

import (
	"context"
	"crypto/md5"
	"fmt"
	"regexp"
	"strings"

	"github.com/hibooboo2/gchat/api"
	"github.com/hibooboo2/gchat/server/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AuthSrv struct {
	db *storage.DB
}
type User struct {
	api.RegisterRequest
}

func New(db *storage.DB) *AuthSrv {
	return &AuthSrv{db: db}
}

func (a *AuthSrv) ValidToken(token string) (string, bool) {
	vals := strings.SplitN(token, "@", 2)
	if len(vals) != 2 {
		return "", false
	}

	u, err := a.db.GetUser(vals[0])
	if err != nil {
		return "", false
	}

	if sumString(u.Username+u.Password) != vals[1] {
		return "", false
	}
	return u.Username, true
}

func (a *AuthSrv) Login(ctx context.Context, req *api.LoginRequest) (*api.LoginResponse, error) {
	u, err := a.db.GetUser(req.Username)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	if u == nil || u.Password != req.Password {
		return nil, status.Errorf(codes.Unauthenticated, "invalid username or password")
	}
	return &api.LoginResponse{Token: req.Username + "@" + sumString(req.Username+req.Password)}, nil
}

func (a *AuthSrv) Register(ctx context.Context, req *api.RegisterRequest) (*api.RegisterResponse, error) {
	if len(req.Password) < 2 {
		return nil, status.Error(codes.InvalidArgument, "password is not longer than 2 characters")
	}
	r := regexp.MustCompile(`[0-9A-Za-z_.]`)

	if !r.Match([]byte(req.Username)) {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("username does not match regex: %s", r.String()))
	}

	return &api.RegisterResponse{}, a.db.SaveUser(req)
}

func sumString(val string) string {
	h := md5.New()
	h.Write([]byte(val))
	return fmt.Sprintf("%x", h.Sum(nil))
}
