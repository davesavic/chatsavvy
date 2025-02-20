package chatsavvy

import (
	"context"
	"log/slog"
	"os"
	"strings"

	"github.com/davesavic/chatsavvy/migrations"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func MigrateDatabase() {
	client, err := mongo.Connect(options.Client().ApplyURI(os.Getenv("MONGODB_URI")))
	if err != nil {
		panic(err)
	}
	defer func() {
		if err = client.Disconnect(context.Background()); err != nil {
			panic(err)
		}
	}()

	err = client.Ping(context.Background(), nil)
	if err != nil {
		panic(err)
	}

	if len(os.Args) < 2 {
		panic("Please provide a direction")
	}

	direction := strings.ToLower(os.Args[1])
	if direction != "up" && direction != "down" {
		panic("Invalid direction. Please provide either 'up' or 'down'")
	}

	err = migrations.Run(client, direction)
	if err != nil {
		panic(err)
	}

	slog.Info("Migration completed")
}
