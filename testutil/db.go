package testutil

import (
	"testing"

	"github.com/davesavic/chatsavvy/migrations"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func MustConnectMongoDB(t *testing.T, dbURI string) *mongo.Client {
	t.Helper()

	client, err := mongo.Connect(options.Client().ApplyURI(dbURI))
	if err != nil {
		t.Fatal(err)
	}

	migrations.Run(client, "down")
	migrations.Run(client, "up")

	return client
}
