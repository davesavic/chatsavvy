package migrations

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"sort"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type Migration struct {
	Timestamp int64
	Up        func(ctx context.Context, db *mongo.Database) error
	Down      func(ctx context.Context, db *mongo.Database) error
}

var MigrationCollection = "migrations"

var Migrations = []Migration{
	{Timestamp: 1739673768, Up: Up1739673768, Down: Down1739673768},
	{Timestamp: 1770967803, Up: Up1770967803, Down: Down1770967803},
}

func Run(client *mongo.Client, direction string) error {
	if direction != "up" && direction != "down" {
		return fmt.Errorf("invalid direction: %s. Please provide either 'up' or 'down'.", direction)
	}

	mgs := Migrations

	sort.Slice(mgs, func(i, j int) bool {
		if direction == "up" {
			return mgs[i].Timestamp < mgs[j].Timestamp
		} else {
			return mgs[i].Timestamp > mgs[j].Timestamp
		}
	})

	db := client.Database("chatsavvy")

	appliedMigrations, err := getAppliedMigrations(context.Background(), db)
	if err != nil {
		return err
	}

	for _, mg := range mgs {
		if direction == "up" && appliedMigrations[mg.Timestamp] {
			slog.Info("Migration already applied", "timestamp", mg.Timestamp)
			continue
		}

		if direction == "down" && !appliedMigrations[mg.Timestamp] {
			slog.Info("Migration not applied", "timestamp", mg.Timestamp)
			continue
		}

		var err error
		if direction == "up" {
			err = mg.Up(context.Background(), db)
		} else {
			err = mg.Down(context.Background(), db)
		}
		if err != nil {
			panic(err)
		}

		if direction == "up" {
			_, err = db.Collection(MigrationCollection).InsertOne(context.Background(), bson.M{"timestamp": mg.Timestamp})
			if err != nil {
				panic(err)
			}

			slog.Info("Migration applied", "timestamp", mg.Timestamp)
		}

		if direction == "down" {
			_, err = db.Collection(MigrationCollection).DeleteOne(context.Background(), bson.M{"timestamp": mg.Timestamp})
			if err != nil {
				panic(err)
			}

			slog.Info("Migration reverted", "timestamp", mg.Timestamp)
		}
	}
	return nil
}

func getAppliedMigrations(ctx context.Context, db *mongo.Database) (map[int64]bool, error) {
	appliedMigrations := make(map[int64]bool)

	collections, err := db.ListCollectionNames(ctx, bson.M{})
	if err != nil {
		return nil, err
	}

	collectionExists := slices.Contains(collections, MigrationCollection)
	if !collectionExists {
		err = db.CreateCollection(ctx, MigrationCollection)
		if err != nil {
			return nil, err
		}
	}

	cursor, err := db.Collection(MigrationCollection).Find(ctx, bson.M{})
	if err != nil {
		if errors.Is(err, mongo.ErrNilDocument) {
			slog.Info("No migrations found")
			return appliedMigrations, nil
		}

		return nil, err
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var record struct {
			Timestamp int64 `bson:"timestamp"`
		}
		if err := cursor.Decode(&record); err != nil {
			return nil, err
		}

		appliedMigrations[record.Timestamp] = true
	}

	return appliedMigrations, nil
}
