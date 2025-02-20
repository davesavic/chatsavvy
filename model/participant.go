package model

type Participant struct {
	ParticipantID string         `bson:"participant_id"`
	Metadata      map[string]any `bson:"metadata"`
}
