package data

import "github.com/go-playground/validator/v10"

type CreateParticipant struct {
	ParticipantID string         `validate:"required,min=1,max=100" bson:"participant_id"`
	Metadata      map[string]any `validate:"omitempty" bson:"metadata"`
}

func (c CreateParticipant) Validate() error {
	return validator.New().Struct(c)
}
