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
		Participants: []csdata.AddParticipant{
			{ParticipantID: "1234567890"},
			{ParticipantID: "0987654321"},
			{ParticipantID: "1357924680", Metadata: map[string]any{"business_id": "1234567890"}},
		},
		Metadata: map[string]any{},
	})

	cs.Conversation.Paginate(context.Background(), csdata.PaginateConversations{
		ParticipantID: "1234567890",
		Page:          1,
		PerPage:       10,
	})

	cs.Conversation.Find(context.Background(), "1234567890")

	cs.Conversation.AddParticipant(context.Background(), "1234567890", csdata.AddParticipant{
		ParticipantID: "0987654321",
		Metadata: map[string]any{
			"business_id": "1234567891",
		},
	})

	cs.Conversation.DeleteParticipant(context.Background(), "1234567890", csdata.DeleteParticipant{})

	cs.Conversation.ParticipantExists(context.Background(), "1234567890", csdata.ParticipantExists{
		ParticipantID: "1234567890",
		Metadata:      map[string]any{"business_id": "1234567890"},
	})

	cs.Conversation.FindByParticipants(context.Background(), csdata.FindByParticipants{
		Participants: []csdata.FindParticipant{
			{ParticipantID: "1234567890"},
			{ParticipantID: "0987654321", Metadata: map[string]any{"business_id": "1234567890"}},
		},
	})

	cs.Conversation.FindByMetadata(context.Background(), csdata.FindByMetadata{
		Metadata:       map[string]any{"business_id": "1234567890"},
		MatchMode:      csdata.MetadataMatchModeKeyValue,
		Page:           1,
		PerPage:        10,
		IncludeDeleted: false,
	})

	cs.Message.Create(context.Background(), "1234567890", csdata.CreateMessage{
		Kind: "general",
		Sender: csdata.MessageSender{
			ParticipantID: "1234567890",
		},
		Content: "Hello, World!",
	})

	cs.Message.Paginate(context.Background(), csdata.PaginateMessages{
		ConversationID: "1234567890",
		Page:           1,
		PerPage:        10,
	})

	cs.Message.LoadMessages(context.Background(), csdata.LoadMessages{
		ConversationID: "1234567890",
		LastMessageID:  nil,
		PerPage:        10,
	})

	cs.Message.ToggleReaction(context.Background(), csdata.ToggleReaction{
		MessageID: "0987654321",
		Emoji:     ":thumbsup:",
		Participant: csdata.ReactionParticipant{
			ParticipantID: "1234567890",
			Metadata: map[string]any{
				"business_id": "1234567891",
			},
		},
	})

	// cs.Conversation.Delete(context.Background(), "1234567890")
}
