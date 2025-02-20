package migrations

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

func Up1739673768(ctx context.Context, db *mongo.Database) error {
	if err := db.CreateCollection(ctx, "conversations"); err != nil {
		return err
	}

	db.Collection("conversations").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: "participants.participant_id", Value: 1},
			{Key: "updated_at", Value: -1},
		},
	})

	if err := db.CreateCollection(ctx, "messages"); err != nil {
		return err
	}

	return nil
}

func Down1739673768(ctx context.Context, db *mongo.Database) error {
	if err := db.Collection("messages").Drop(ctx); err != nil {
		return err
	}

	if err := db.Collection("conversations").Drop(ctx); err != nil {
		return err
	}

	return nil
}
