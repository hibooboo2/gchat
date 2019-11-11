package auth

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"regexp"
	"strings"
	"time"

	"github.com/hibooboo2/gchat/api"
	"github.com/hibooboo2/gchat/server/storage"
	"github.com/hibooboo2/gchat/utils"
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
	token = utils.Decrypt(token, encryptionKey)
	vals := strings.SplitN(token, "@", 2)
	if len(vals) != 2 {
		return "", false
	}

	u, err := a.db.GetUser(vals[0])
	if err != nil {
		return "", false
	}
	t, err := time.Parse(time.RFC3339Nano, vals[1])
	if err != nil {
		log.Println("err: invalid time format in auth token")
		return "", false
	}
	if time.Now().After(t) {
		log.Println("err: token is expired")
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
	return &api.LoginResponse{Token: genToken(req)}, nil
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

func genToken(req *api.LoginRequest) string {
	token := req.Username + "@" + time.Now().Add(time.Hour*24*7).Format(time.RFC3339Nano)
	token = utils.Encrypt(token, encryptionKey)
	log.Println(token)
	return token
}

var encryptionKey string

func init() {
	rand.Seed(time.Now().UnixNano())
	key := make([]byte, 100)
	n, err := rand.Read(key)
	if err != nil {
		panic(err)
	}
	encryptionKey = string(key[:n])
}
