package model

import "time"

type Participant struct {
	ParticipantID string         `bson:"participant_id"`
	Metadata      map[string]any `bson:"metadata"`
	DeletedAt     *time.Time     `bson:"deleted_at"`
}
