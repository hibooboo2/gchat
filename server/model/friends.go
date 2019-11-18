package model

import (
	"time"

	"github.com/gofrs/uuid"
	"github.com/jinzhu/gorm"
)

type User struct {
	BaseTimes
	Username        string `json:"username" gorm:"type:VARCHAR(36);primary_key;column:username"`
	Password  string
	Email     string
	FirstName string
	LastName  string
	Status    string
	IsOnline  bool
}

func (User) TableName() string {
	return "users"
}

type Message struct {
	Base
	Data   string
	From string 
	To   string
}

type RoomMessage struct {
	Base
	Data string
	From string
	To   string
}

type Room struct {
	Base
	Name string
}

type Base struct {
	ID        string `json:"id" gorm:"type:VARCHAR(36);primary_key;column:id"`
	BaseTimes
}
type BaseTimes struct{
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time `sql:"index"`
}

// BeforeCreate will set a UUID rather than numeric ID.
func (base *Base) BeforeCreate(scope *gorm.Scope) error {
	if base.ID == "" {
		result, _ := uuid.NewV4()
		base.ID = result.String()
	}
	return scope.SetColumn("ID", base.ID)
}

type Friend struct {
	Base
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
