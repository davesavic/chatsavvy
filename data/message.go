package data

import (
	"errors"

	"github.com/go-playground/validator/v10"
)

type MessageSender struct {
	ParticipantID string         `validate:"required,min=1,max=100" bson:"participant_id"`
	Metadata      map[string]any `validate:"omitempty" bson:"metadata"`
}

type CreateAttachment struct {
	Kind     string         `validate:"required,min=1,max=100" bson:"kind"`
	Metadata map[string]any `validate:"omitempty" bson:"metadata"`
}

type CreateMessage struct {
	Kind        string             `validate:"required,oneof=general system" bson:"kind"`
	Sender      MessageSender      `validate:"required" bson:"sender"`
	Content     string             `validate:"omitempty,max=5000" bson:"content"`
	Attachments []CreateAttachment `validate:"omitempty,max=10,dive" bson:"attachments"`
}

func (c CreateMessage) Validate() error {
	if err := validator.New().Struct(c); err != nil {
		return err
	}

	if c.Content == "" && len(c.Attachments) == 0 {
		return errors.New("message must have content or at least one attachment")
	}

	return nil
}

type PaginateMessages struct {
	ConversationID string `validate:"required,min=1,max=100" bson:"conversation_id"`
	Page           uint   `validate:"required,min=1" bson:"page"`
	PerPage        uint   `validate:"required,min=1,max=100" bson:"per_page"`
}

func (c PaginateMessages) Validate() error {
	return validator.New().Struct(c)
}

type LoadMessages struct {
	ConversationID string  `validate:"required,min=1,max=100" bson:"conversation_id"`
	LastMessageID  *string `validate:"omitempty,min=1,max=100" bson:"last_message_id"`
	PerPage        uint    `validate:"required,min=1,max=100" bson:"per_page"`
}

func (c LoadMessages) Validate() error {
	return validator.New().Struct(c)
}

type ReactionParticipant struct {
	ParticipantID string         `validate:"required,min=1,max=100" bson:"participant_id"`
	Metadata      map[string]any `validate:"omitempty" bson:"metadata"`
}

type ToggleReaction struct {
	MessageID   string              `validate:"required,min=1,max=100" bson:"message_id"`
	Emoji       string              `validate:"required,min=1,max=100" bson:"emoji"`
	Participant ReactionParticipant `validate:"required" bson:"participant"`
}

func (c ToggleReaction) Validate() error {
	return validator.New().Struct(c)
}
