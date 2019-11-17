package storage

import (
	"log"
	"sync"

	"github.com/hibooboo2/gchat/api"
	"github.com/hibooboo2/gchat/server/model"
	"github.com/hibooboo2/gchat/utils"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (d *DB) GetUserID(username string) (uint, error) {
	u := model.User{}
	err := d.db.Select([]string{"id"}).Find(&u, "username = ?", username).Error
	if err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return 0, status.Errorf(codes.NotFound, "user not found")
		}
		return 0, errors.Wrapf(err, "failed to get user: %s", username)
	}
	return u.ID, nil
}

func (d *DB) GetUser(username string) (*model.User, error) {
	u := model.User{}
	err := d.db.Find(&u, "username = ?", username).Error
	if err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return nil, nil
		}
		return nil, errors.Wrapf(err, "failed to get user: %s", username)
	}
	return &u, nil
}

func (d *DB) SaveUser(u *api.RegisterRequest) error {
	err := d.db.First(&model.User{}, "username = ?", u.Username).Error
	if err != nil && !gorm.IsRecordNotFoundError(err) {
		return errors.Wrapf(err, "failed to check for existing user")
	}

	if err == nil {
		return status.Errorf(codes.AlreadyExists, "username already taken")
	}

	usr := model.User{
		Username:  u.Username,
		Password:  utils.Hash(u.Password),
		LastName:  u.LastName,
		FirstName: u.FirstName,
		Email:     u.Email,
		Status:    "offline",
	}
	err = d.db.Save(&usr).Error
	if err != nil {
		return status.Errorf(codes.Internal, "failed to save user: %s", err.Error())
	}
	err = d.db.Save(&model.Friend{UserA: u.Username, UserB: u.Username}).Error
	if err != nil {
		return status.Errorf(codes.Internal, "failed to save friend request: %v", err)
	}
	return nil
}

func (d *DB) UserOnline(username string, online bool, usersOnline sync.Map) error {
	st := "online"
	if !online {
		st = "offline"
	}

	err := d.db.Model(&model.User{}).Where(`username = ?`, username).Update("status", st).Error
	log.Printf("trace: user online %s %v", username, online)
	if err != nil {
		return errors.Wrapf(err, "failed to update online status for user %s", username)
	}
	l, err := d.AllFriends(username)
	if err != nil {
		return errors.Wrapf(err, "failed to get all friends to send status for user %s", username)
	}

	friends := map[string]struct{}{}
	for _, friend := range l.Friends {
		friends[friend.Username] = struct{}{}
	}
	log.Printf("trace: got all friends to send status for user %v", friends)

	status := &api.FriendStatus{Username: username, Status: st}
	usersOnline.Range(func(key interface{}, value interface{}) bool {
		log.Printf("Checking key: %v", key)
		friendName, ok := key.(string)
		if !ok {
			return true
		}

		if _, ok := friends[friendName]; !ok {
			return true
		}

		stream, ok := value.(api.Friends_StatusServer)
		if !ok {
			return true
		}
		err := stream.Send(status)
		if err != nil {
			log.Printf("err: failed to send status to: %s", username)
			return true
		}
		log.Printf("info: Sent status to: %s", username)

		return true
	})

	return nil
}
