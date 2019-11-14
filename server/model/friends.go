package model

import "github.com/jinzhu/gorm"

type User struct {
	gorm.Model
	Username  string
	Password  string
	Email     string
	FirstName string
	LastName  string
	Status    string
}

type Message struct {
	gorm.Model
	Data   string
	FromID uint
	ToID   uint
}

type Friend struct {
	gorm.Model
	UserA string
	UserB string
}

func (Friend) TableName() string {
	return "friends"
}

type FriendRequest struct {
	Requestor string
	User      string
}

func (FriendRequest) TableName() string {
	return "friend_requests"
}
