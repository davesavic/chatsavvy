package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/davesavic/chatsavvy/data"
	"github.com/davesavic/chatsavvy/model"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type Message struct {
	db           *mongo.Database
	conversation *Conversation
}

func NewMessage(db *mongo.Database, conversation *Conversation) *Message {
	return &Message{
		db:           db,
		conversation: conversation,
	}
}

func (m Message) Create(ctx context.Context, conversationID string, d data.CreateMessage) (*model.Message, error) {
	if err := d.Validate(); err != nil {
		return nil, err
	}

	conversation, err := m.conversation.Find(ctx, conversationID)
	if err != nil || conversation == nil {
		return nil, fmt.Errorf("failed to fetch the conversation: %w", err)
	}

	now := time.Now()
	bsonNow := bson.NewDateTimeFromTime(now)

	res, err := m.db.Collection("messages").InsertOne(ctx, bson.M{
		"conversation_id": conversation.ID.Hex(),
		"sender":          d.Sender,
		"kind":            d.Kind,
		"content":         d.Content,
		"created_at":      bsonNow,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to insert message: %w", err)
	}

	var message model.Message
	err = m.db.Collection("messages").FindOne(ctx, bson.M{"_id": res.InsertedID}).Decode(&message)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch the message: %w", err)
	}

	err = m.conversation.UpdateLastMessage(ctx, conversationID, message)
	if err != nil {
		return nil, fmt.Errorf("failed to touch the conversation: %w", err)
	}

	return &message, nil
}

func (m Message) Paginate(ctx context.Context) error {
	return nil
}

func (m Message) ToggleReaction(ctx context.Context) error {
	return nil
}
