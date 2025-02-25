package model

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type Conversation struct {
	ID           bson.ObjectID  `bson:"_id"`
	Participants []Participant  `bson:"participants"`
	Metadata     map[string]any `bson:"metadata"`
	LastMessage  *Message       `bson:"last_message"`
	CreatedAt    time.Time      `bson:"created_at"`
	UpdatedAt    time.Time      `bson:"updated_at"`
}
