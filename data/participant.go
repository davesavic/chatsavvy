package data

import "github.com/go-playground/validator/v10"

type AddParticipant struct {
	ParticipantID string         `validate:"required,min=1,max=100" bson:"participant_id"`
	Metadata      map[string]any `validate:"omitempty" bson:"metadata"`
}

func (c AddParticipant) Validate() error {
	return validator.New().Struct(c)
}

type DeleteParticipant struct {
	ParticipantID string         `validate:"required,min=1,max=100" bson:"participant_id"`
	Metadata      map[string]any `validate:"omitempty" bson:"metadata"`
}

func (c DeleteParticipant) Validate() error {
	return validator.New().Struct(c)
}

type ParticipantExists struct {
	ParticipantID string         `validate:"required,min=1,max=100" bson:"participant_id"`
	Metadata      map[string]any `validate:"omitempty" bson:"metadata"`
}

func (c ParticipantExists) Validate() error {
	return validator.New().Struct(c)
}
