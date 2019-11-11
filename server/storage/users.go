package storage

import (
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
	}
	return d.db.Save(&usr).Error
}
