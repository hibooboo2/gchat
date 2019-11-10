package storage

import (
	"github.com/hibooboo2/gchat/api"
	"github.com/hibooboo2/gchat/server/model"
	"github.com/pkg/errors"
)

func (d *DB) SaveMessage(m *api.Message, from string) error {
	to, err := d.GetUserID(m.To)
	if err != nil {
		return errors.Wrapf(err, "failed to get to user ID")
	}
	fromID, err := d.GetUserID(from)
	if err != nil {
		return errors.Wrapf(err, "failed to get from user ID")
	}
	msg := model.Message{
		Data:   m.Data,
		FromID: fromID,
		ToID:   to,
	}

	return d.db.Save(&msg).Error
}

func (d *DB) GetMessages(username string, from string) (*api.MessageList, error) {
	userA, err := d.GetUserID(username)
	if err != nil {
		return nil, err
	}
	userB, err := d.GetUserID(from)
	if err != nil {
		return nil, err
	}
	ids := map[uint]string{}
	ids[userA] = username
	ids[userB] = from

	msgs := []model.Message{}
	err = d.db.Debug().Find(&msgs, "((to_id = ? AND from_id = ?) OR (from_id = ? AND to_id = ?))", userA, userB, userA, userB).Error
	if err != nil {
		return nil, errors.Wrap(err, "failed to get messages between users")
	}
	messages := api.MessageList{}
	for _, m := range msgs {
		messages.Messages = append(messages.Messages, &api.Message{Data: m.Data, From: ids[m.FromID], To: ids[m.ToID]})
	}
	return &messages, nil
}
