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

type Conversation struct {
	db *mongo.Database
}

func NewConversation(db *mongo.Database) *Conversation {
	return &Conversation{db: db}
}

func (c Conversation) AddParticipant(ctx context.Context, conversationID string, d data.CreateParticipant) (*model.Conversation, error) {
	if err := d.Validate(); err != nil {
		return nil, fmt.Errorf("failed to validate create participant data: %w", err)
	}

	thisConversation, err := c.Find(ctx, conversationID)
	if err != nil || thisConversation == nil {
		return nil, fmt.Errorf("failed to fetch the conversation: %w", err)
	}

	thisConversationParticipants := thisConversation.Participants
	participantsToCheck := make([]data.CreateParticipant, 0, len(thisConversationParticipants)+1)
	participantsToCheck = append(participantsToCheck, d)
	for _, p := range thisConversationParticipants {
		participantsToCheck = append(participantsToCheck, data.CreateParticipant{
			ParticipantID: p.ParticipantID,
			Metadata:      p.Metadata,
		})
	}

	exists, existingConversation, err := c.conversationWithParticipantsExists(ctx, participantsToCheck)
	if err != nil {
		return nil, fmt.Errorf("failed to check if conversation exists: %w", err)
	}
	if exists {
		return existingConversation, nil
	}

	conversationIDHex, err := bson.ObjectIDFromHex(conversationID)
	if err != nil {
		return nil, fmt.Errorf("failed to parse conversation id: %w", err)
	}

	filter := bson.M{
		"_id": conversationIDHex,
	}

	update := bson.M{
		"$push": bson.M{
			"participants": d,
		},
		"$set": bson.M{
			"updated_at": bson.NewDateTimeFromTime(time.Now()),
		},
	}

	res, err := c.db.Collection("conversations").UpdateOne(ctx, filter, update)
	if err != nil {
		return nil, fmt.Errorf("failed to add participant: %w", err)
	}

	if res.MatchedCount == 0 {
		return nil, fmt.Errorf("conversation not found")
	}

	var conversation model.Conversation
	err = c.db.Collection("conversations").FindOne(ctx, filter).Decode(&conversation)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch conversation: %w", err)
	}

	return &conversation, nil
}

func (c Conversation) DeleteParticipant(ctx context.Context, conversationID, participantID string) (*model.Conversation, error) {
	conversationIDHex, err := bson.ObjectIDFromHex(conversationID)
	if err != nil {
		return nil, fmt.Errorf("failed to parse conversation id: %w", err)
	}

	filter := bson.M{
		"_id": conversationIDHex,
	}

	now := time.Now()
	update := bson.M{
		"$set": bson.M{
			"participants.$[participant].deleted_at": bson.NewDateTimeFromTime(now),
			"updated_at":                             bson.NewDateTimeFromTime(now),
		},
	}

	arrayFilters := []any{
		bson.M{
			"participant.participant_id": participantID,
		},
	}

	res, err := c.db.Collection("conversations").UpdateOne(ctx, filter, update, options.UpdateOne().SetArrayFilters(arrayFilters))
	if err != nil {
		return nil, fmt.Errorf("failed to delete participant: %w", err)
	}

	if res.MatchedCount == 0 {
		return nil, fmt.Errorf("conversation not found")
	}

	var conversation model.Conversation
	err = c.db.Collection("conversations").FindOne(ctx, filter).Decode(&conversation)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch conversation: %w", err)
	}

	return &conversation, nil
}

func (c Conversation) conversationWithParticipantsExists(ctx context.Context, participants []data.CreateParticipant) (bool, *model.Conversation, error) {
	participantCount := len(participants)

	participantsFilters := make([]bson.M, 0, participantCount)
	for _, p := range participants {
		participantsFilters = append(participantsFilters, bson.M{
			"$elemMatch": bson.M{
				"participant_id": p.ParticipantID,
				"metadata":       p.Metadata,
			},
		})
	}

	filter := bson.M{
		"$and": []bson.M{
			{
				"$expr": bson.M{
					"$eq": []interface{}{
						bson.M{"$size": "$participants"},
						participantCount,
					},
				},
			},
			{
				"participants": bson.M{
					"$all": participantsFilters,
				},
			},
		},
	}

	var existingConversation model.Conversation
	err := c.db.Collection("conversations").FindOne(ctx, filter).Decode(&existingConversation)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return false, nil, nil
		}
		return false, nil, fmt.Errorf("failed to fetch existing conversation: %w", err)
	}

	return true, &existingConversation, nil
}

func (c Conversation) Create(ctx context.Context, d data.CreateConversation) (*model.Conversation, error) {
	if err := d.Validate(); err != nil {
		return nil, fmt.Errorf("failed to validate create conversation data: %w", err)
	}

	exists, existingConversation, err := c.conversationWithParticipantsExists(ctx, d.Participants)
	if err != nil {
		return nil, fmt.Errorf("failed to check if conversation exists: %w", err)
	}
	if exists {
		return existingConversation, nil
	}

	res, err := c.db.Collection("conversations").InsertOne(ctx, bson.M{
		"participants": d.Participants,
		"metadata":     d.Metadata,
		"created_at":   bson.NewDateTimeFromTime(time.Now()),
		"updated_at":   bson.NewDateTimeFromTime(time.Now()),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create conversation: %w", err)
	}

	var conversation model.Conversation
	err = c.db.Collection("conversations").FindOne(ctx, bson.M{"_id": res.InsertedID}).Decode(&conversation)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch raw conversation: %w", err)
	}

	return &conversation, nil
}

func (c Conversation) Find(ctx context.Context, id string) (*model.Conversation, error) {
	obID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("failed to parse conversation id: %w", err)
	}

	var conversation model.Conversation
	err = c.db.Collection("conversations").FindOne(ctx, bson.M{"_id": obID}).Decode(&conversation)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to fetch conversation: %w", err)
	}

	return &conversation, nil
}

func (c Conversation) Paginate(ctx context.Context, d data.PaginateConversations) ([]model.Conversation, uint, error) {
	if err := d.Validate(); err != nil {
		return nil, 0, fmt.Errorf("failed to validate paginate conversations data: %w", err)
	}

	filter := bson.M{
		"participants.participant_id": d.ParticipantID,
	}

	total, err := c.db.Collection("conversations").CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count conversations: %w", err)
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "updated_at", Value: -1}}).
		SetSkip(int64(d.Page-1) * int64(d.PerPage)).
		SetLimit(int64(d.PerPage))

	cursor, err := c.db.Collection("conversations").Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to fetch conversations: %w", err)
	}
	defer cursor.Close(ctx)

	var conversations []model.Conversation
	if err := cursor.All(ctx, &conversations); err != nil {
		return nil, 0, fmt.Errorf("failed to decode conversations: %w", err)
	}

	return conversations, uint(total), nil
}
