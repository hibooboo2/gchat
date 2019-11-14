package storage

import (
	"github.com/hibooboo2/gchat/server/model"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/pkg/errors"
)

type DB struct {
	db *gorm.DB
}

func New(fileLoc string) (*DB, error) {
	db, err := gorm.Open("sqlite3", fileLoc)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open db")
	}
	db.AutoMigrate(&model.Message{}, &model.User{}, &model.Friend{}, &model.FriendRequest{})
	return &DB{db}, nil
}
