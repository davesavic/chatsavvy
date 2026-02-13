package repository

import (
	"context"
	"fmt"
	"slices"
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

// Create creates a new message in the conversation.
// It returns the created message or an error.
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
		"attachments":     d.Attachments,
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

// Paginate fetches messages in the conversation.
// It returns the messages and the total number of messages in the conversation or an error.
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

// LoadMessages fetches messages in the conversation.
// It differs from Paginate in that it fetches messages older than the last message id provided.
// If the last message id is nil, it fetches the latest messages.
// It returns the messages or an error.
func (m Message) LoadMessages(ctx context.Context, d data.LoadMessages) ([]model.Message, error) {
	if err := d.Validate(); err != nil {
		return nil, err
	}

	conv, err := m.conversation.Find(ctx, d.ConversationID)
	if err != nil || conv == nil {
		return nil, fmt.Errorf("failed to fetch the conversation: %w", err)
	}

	filter := bson.M{
		"conversation_id": d.ConversationID,
	}

	if d.LastMessageID != nil {
		messageObID, err := bson.ObjectIDFromHex(*d.LastMessageID)
		if err != nil {
			return nil, fmt.Errorf("failed to parse the last message id: %w", err)
		}

		filter["_id"] = bson.M{"$lt": messageObID}
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

// ToggleReaction toggles a reaction on the message for the participant.
// It returns the updated message or an error.
func (m Message) ToggleReaction(ctx context.Context, d data.ToggleReaction) (*model.Message, error) {
	if err := d.Validate(); err != nil {
		return nil, err
	}
	messageObID, err := bson.ObjectIDFromHex(d.MessageID)
	if err != nil {
		return nil, fmt.Errorf("failed to parse the message id: %w", err)
	}
	var message model.Message
	err = m.db.Collection("messages").FindOne(ctx, bson.M{"_id": messageObID}).Decode(&message)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch the message: %w", err)
	}

	reactionIndex := slices.IndexFunc(message.Reactions, func(r model.Reaction) bool {
		return r.Emoji == d.Emoji
	})

	if reactionIndex != -1 {
		reaction := message.Reactions[reactionIndex]

		participantIndex := slices.IndexFunc(reaction.Participants, func(p model.ReactionParticipant) bool {
			return p.ParticipantID == d.Participant.ParticipantID && mapsEqual(p.Metadata, d.Participant.Metadata)
		})

		if participantIndex == -1 {
			reaction.Participants = append(reaction.Participants, model.ReactionParticipant{
				ParticipantID: d.Participant.ParticipantID,
				Metadata:      d.Participant.Metadata,
			})
		} else {
			reaction.Participants = slices.Delete(reaction.Participants, participantIndex, participantIndex+1)
		}

		message.Reactions[reactionIndex] = reaction
	}

	if reactionIndex == -1 {
		reaction := model.Reaction{
			Emoji: d.Emoji,
			Participants: []model.ReactionParticipant{
				{
					ParticipantID: d.Participant.ParticipantID,
					Metadata:      d.Participant.Metadata,
				},
			},
		}
		message.Reactions = append(message.Reactions, reaction)
	}

	message.Reactions = slices.DeleteFunc(message.Reactions, func(r model.Reaction) bool {
		return len(r.Participants) == 0
	})

	update := bson.M{
		"$set": bson.M{
			"reactions": message.Reactions,
		},
	}
	res, err := m.db.Collection("messages").UpdateOne(ctx, bson.M{"_id": messageObID}, update)
	if err != nil {
		return nil, fmt.Errorf("failed to update message: %w", err)
	}
	if res.MatchedCount == 0 {
		return nil, fmt.Errorf("message not found")
	}
	return &message, nil
}

func mapsEqual(a, b map[string]any) bool {
	if len(a) != len(b) {
		return false
	}
	for key, va := range a {
		vb, ok := b[key]
		if !ok || !valuesEqual(va, vb) {
			return false
		}
	}
	return true
}

func valuesEqual(a, b any) bool {
	switch va := a.(type) {
	case int:
		vb, ok := b.(int)
		return ok && va == vb
	case float64:
		vb, ok := b.(float64)
		return ok && va == vb
	case string:
		vb, ok := b.(string)
		return ok && va == vb
	case bool:
		vb, ok := b.(bool)
		return ok && va == vb
	case map[string]any:
		vb, ok := b.(map[string]any)
		return ok && mapsEqual(va, vb)
	default:
		return false
	}
}
