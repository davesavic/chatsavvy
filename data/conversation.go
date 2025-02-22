package data

import "github.com/go-playground/validator/v10"

type CreateConversation struct {
	Participants []AddParticipant `validate:"required,min=2,max=10"`
	Metadata     map[string]any   `validate:"omitempty"`
}

func (c CreateConversation) Validate() error {
	return validator.New().Struct(c)
}

type PaginateConversations struct {
	ParticipantID string `validate:"required,min=1,max=100"`
	Page          uint   `validate:"required,min=1"`
	PerPage       uint   `validate:"required,min=1,max=100"`
}

func (p PaginateConversations) Validate() error {
	return validator.New().Struct(p)
}
