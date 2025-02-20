package main

import (
	"log/slog"
	"os"

	"github.com/davesavic/chatsavvy/migrations"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func main() {
	args := os.Args
	if len(args) < 2 {
		panic("Please provide a direction (up or down)")
	}

	direction := args[1]
	if direction != "up" && direction != "down" {
		panic("Invalid direction. Please provide either 'up' or 'down'")
	}

	client, err := mongo.Connect(options.Client().ApplyURI(os.Getenv("MONGODB_URI")))
	if err != nil {
		panic(err)
	}
	defer func() {
		if err = client.Disconnect(nil); err != nil {
			panic(err)
		}
	}()

	err = client.Ping(nil, nil)
	if err != nil {
		panic(err)
	}

	err = migrations.Run(client, direction)
	if err != nil {
		panic(err)
	}

	slog.Info("Migration completed")
}
