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

func (d *DB) UpdateStatus(username string, online bool, status string, usersOnline sync.Map) error {
	u := model.User{}
	q := d.db.Model(&model.User{}).Where(`username = ?`, username).Update("is_online", online)
	log.Printf("info: status [%s] [%v] [%s]", username, online, status)
	if status != "" {
		log.Printf("info: Had status [%s] [%v] [%s]", username, online, status)
		q.Update("status", status)
	}

	err := q.Select([]string{"username", "status", "is_online"}).First(&u, "username = ?", username).Error
	if err != nil {
		return errors.Wrapf(err, "failed to update online status for user %s", username)
	}

	log.Printf("warn: %s %s %v", u.Username, u.Status, u.IsOnline)

	return d.sendStatusUpdates(u.Username, u.Status, u.IsOnline, usersOnline)
}

func (d *DB) sendStatusUpdates(username string, st string, online bool, usersOnline sync.Map) error {
	l, err := d.AllFriends(username)
	if err != nil {
		return errors.Wrapf(err, "failed to get all friends to send status for user %s", username)
	}

	log.Printf("trace: got all friends to send status for user %v", l.Friends)

	status := &api.FriendStatus{Username: username, Status: st, Online: online}

	for _, friend := range l.Friends {
		value, ok := usersOnline.Load(friend.Username)
		if !ok {
			continue
		}

		stream, ok := value.(api.Friends_StatusServer)
		if !ok {
			continue
		}

		err := stream.Send(status)
		if err != nil {
			log.Printf("err: failed to send status to: %s", username)
			continue
		}
		log.Printf("info: Sent status to: %s", username)

	}
	return nil
}
