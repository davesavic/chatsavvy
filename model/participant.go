package model

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type Participant struct {
	ParticipantID     string         `bson:"participant_id"`
	Metadata          map[string]any `bson:"metadata"`
	DeletedAt         *time.Time     `bson:"deleted_at"`
	LastReadMessageID *bson.ObjectID `bson:"last_read_message_id"`
	LastReadAt        *time.Time     `bson:"last_read_at"`
}
