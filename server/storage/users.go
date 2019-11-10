package storage

import (
	"log"

	"github.com/hibooboo2/gchat/api"
	"github.com/hibooboo2/gchat/server/model"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (d *DB) GetUserID(username string) (uint, error) {
	u := model.User{}
	err := d.db.Select([]string{"id"}).Find(&u, "username = ?", username).Error
	if err != nil {
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
	log.Println(u)
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
		Password:  u.Password,
		LastName:  u.LastName,
		FirstName: u.FirstName,
		Email:     u.Email,
	}
	return d.db.Save(&usr).Error
}
