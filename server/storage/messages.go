package storage

import (
	"github.com/hibooboo2/gchat/api"
	"github.com/hibooboo2/gchat/server/model"
	"github.com/pkg/errors"
)

func (d *DB) SaveMessage(m *api.Message, from string) error {
	msg := model.Message{
		Data: m.Data,
		From: from,
		To:   m.To,
	}
	return d.db.Save(&msg).Error
}

func (d *DB) GetMessages(username string, from string) (*api.MessageList, error) {
	msgs := []model.Message{}
	err := d.db.Find(&msgs, `(("to" = ? AND "from" = ?) OR ("from" = ? AND "to" = ?))`, username, from, username, from).Error
	if err != nil {
		return nil, errors.Wrap(err, "failed to get messages between users")
	}
	messages := api.MessageList{}
	for _, m := range msgs {
		messages.Messages = append(messages.Messages, &api.Message{Data: m.Data, From: m.From, To: m.To})
	}
	return &messages, nil
}
