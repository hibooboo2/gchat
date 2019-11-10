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
