package model

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type MessageSender struct {
	ParticipantID string         `bson:"participant_id"`
	Metadata      map[string]any `bson:"metadata"`
}

type Attachment struct {
	Kind     string         `bson:"kind"`
	Metadata map[string]any `bson:"metadata"`
}

type Message struct {
	ID             bson.ObjectID `bson:"_id"`
	ConversationID bson.ObjectID `bson:"conversation_id"`
	Sender         MessageSender `bson:"sender"`
	Kind           string        `bson:"kind"`
	Content        string        `bson:"content"`
	Attachments    []Attachment  `bson:"attachments"`
	Reactions      []Reaction    `bson:"reactions"`
	CreatedAt      time.Time     `bson:"created_at"`
}

type ReactionParticipant struct {
	ParticipantID string         `bson:"participant_id"`
	Metadata      map[string]any `bson:"metadata"`
}

type Reaction struct {
	Emoji        string                `bson:"emoji"`
	Participants []ReactionParticipant `bson:"participants"`
}
