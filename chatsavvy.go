package chatsavvy

import (
	"context"
	"time"

	"github.com/davesavic/chatsavvy/migrations"
	"github.com/davesavic/chatsavvy/repository"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type ChatSavvy struct {
	client *mongo.Client

	Conversation *repository.Conversation
	Message      *repository.Message
}

func New(client *mongo.Client) (*ChatSavvy, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := client.Ping(ctx, nil)
	if err != nil {
		return nil, err
	}

	db := client.Database("chatsavvy")

	return &ChatSavvy{
		client: client,

		Conversation: repository.NewConversation(db),
		Message:      repository.NewMessage(db, repository.NewConversation(db)),
	}, nil
}

func (cs *ChatSavvy) Close() error {
	return cs.client.Disconnect(context.Background())
}

// Migrate runs the migrations in the specified direction (up or down).
// Beware that running migrations in the down direction will delete all data.
func Migrate(client *mongo.Client, direction string) error {
	return migrations.Run(client, direction)
}
