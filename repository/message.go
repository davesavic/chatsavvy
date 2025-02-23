package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/davesavic/chatsavvy/data"
	"github.com/davesavic/chatsavvy/model"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
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

func (m Message) Paginate(ctx context.Context, d data.PaginateMessages) ([]model.Message, uint, error) {
	if err := d.Validate(); err != nil {
		return nil, 0, err
	}

	conv, err := m.conversation.Find(ctx, d.ConversationID)
	if err != nil || conv == nil {
		return nil, 0, fmt.Errorf("failed to fetch the conversation: %w", err)
	}

	skip := (d.Page - 1) * d.PerPage
	opts := options.Find().SetSort(bson.M{"created_at": -1}).SetSkip(int64(skip)).SetLimit(int64(d.PerPage))

	cursor, err := m.db.Collection("messages").Find(ctx, bson.M{"conversation_id": d.ConversationID}, opts)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to fetch messages: %w", err)
	}
	defer cursor.Close(ctx)

	var messages []model.Message
	if err = cursor.All(ctx, &messages); err != nil {
		return nil, 0, fmt.Errorf("failed to decode messages: %w", err)
	}

	total, err := m.db.Collection("messages").CountDocuments(ctx, bson.M{"conversation_id": d.ConversationID})
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count messages: %w", err)
	}

	return messages, uint(total), nil
}

func (m Message) LoadMessages(ctx context.Context, d data.LoadMessages) ([]model.Message, error) {
	if err := d.Validate(); err != nil {
		return nil, err
	}

	conv, err := m.conversation.Find(ctx, d.ConversationID)
	if err != nil || conv == nil {
		return nil, fmt.Errorf("failed to fetch the conversation: %w", err)
	}

	messageObID, err := bson.ObjectIDFromHex(d.LastMessageID)
	if err != nil {
		return nil, fmt.Errorf("failed to parse the last message id: %w", err)
	}

	filter := bson.M{
		"conversation_id": d.ConversationID,
		"_id":             bson.M{"$lt": messageObID},
	}

	opts := options.Find().SetSort(bson.M{"_id": -1}).SetLimit(int64(d.PerPage))

	cursor, err := m.db.Collection("messages").Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch messages: %w", err)
	}
	defer cursor.Close(ctx)

	var messages []model.Message
	if err = cursor.All(ctx, &messages); err != nil {
		return nil, fmt.Errorf("failed to decode messages: %w", err)
	}

	return messages, nil
}

func (m Message) ToggleReaction(ctx context.Context) error {
	return nil
}
