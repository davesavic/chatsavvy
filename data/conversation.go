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

type FindParticipant struct {
	ParticipantID string         `validate:"required,min=1,max=100"`
	Metadata      map[string]any `validate:"omitempty"`
}

type FindByParticipants struct {
	Participants []FindParticipant `validate:"required,min=2"`
}

func (d FindByParticipants) Validate() error {
	return validator.New().Struct(d)
}

type MetadataMatchMode string

const (
	MetadataMatchModeKeyValue MetadataMatchMode = "key_value"
	MetadataMatchModeExact    MetadataMatchMode = "exact"
)

type FindByMetadata struct {
	Metadata       map[string]any    `validate:"required,min=1"`
	MatchMode      MetadataMatchMode `validate:"required,oneof=key_value exact"`
	Page           uint              `validate:"required,min=1"`
	PerPage        uint              `validate:"required,min=1,max=100"`
	IncludeDeleted bool              `validate:"omitempty"`
}

func (d FindByMetadata) Validate() error {
	return validator.New().Struct(d)
}
