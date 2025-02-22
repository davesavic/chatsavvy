package data

import "github.com/go-playground/validator/v10"

type MessageSender struct {
	ParticipantID string         `validate:"required,min=1,max=100" bson:"participant_id"`
	Metadata      map[string]any `validate:"omitempty" bson:"metadata"`
}

type CreateMessage struct {
	Kind    string        `validate:"required,oneof=general system" bson:"kind"`
	Sender  MessageSender `validate:"required" bson:"sender"`
	Content string        `validate:"required,min=1,max=5000" bson:"content"`
}

func (c CreateMessage) Validate() error {
	return validator.New().Struct(c)
}
