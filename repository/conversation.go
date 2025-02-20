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

func (c Conversation) Create(ctx context.Context, d data.CreateConversation) (*model.Conversation, error) {
	if err := d.Validate(); err != nil {
		return nil, fmt.Errorf("failed to validate create conversation data: %w", err)
	}

	participants := d.Participants
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
		if err != mongo.ErrNoDocuments {
			return nil, fmt.Errorf("failed to fetch existing conversation: %w", err)
		}
	} else {
		return &existingConversation, nil
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
