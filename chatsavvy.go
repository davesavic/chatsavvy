package chatsavvy

import (
	"context"
	"time"

	"github.com/davesavic/chatsavvy/repository"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type ChatSavvy struct {
	client *mongo.Client

	Conversation repository.Conversation
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

		Conversation: *repository.NewConversation(db),
	}, nil
}
