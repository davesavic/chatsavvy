package main

import (
	"context"

	"github.com/davesavic/chatsavvy"
	csdata "github.com/davesavic/chatsavvy/data"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func main() {
	client, err := mongo.Connect(options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		panic(err)
	}
	defer client.Disconnect(context.Background())

	cs, err := chatsavvy.New(client)
	if err != nil {
		panic(err)
	}

	cs.Conversation.Create(context.Background(), csdata.CreateConversation{
		Participants: []csdata.CreateParticipant{},
		Metadata:     map[string]any{},
	})
}
