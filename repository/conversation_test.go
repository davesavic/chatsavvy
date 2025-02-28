package repository_test

import (
	"os"
	"testing"
	"time"

	"github.com/davesavic/chatsavvy/data"
	"github.com/davesavic/chatsavvy/model"
	"github.com/davesavic/chatsavvy/repository"
	"github.com/davesavic/chatsavvy/testutil"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestConversationRepository_Create(t *testing.T) {
	client := testutil.MustConnectMongoDB(t, os.Getenv("MONGODB_URI"))
	t.Cleanup(func() {
		client.Disconnect(nil)
	})

	cr := repository.NewConversation(client.Database("chatsavvy"))

	testCases := []struct {
		name     string
		prepData func(t *testing.T) data.CreateConversation
		asserts  func(t *testing.T, conv *model.Conversation, err error)
	}{
		{
			name: "valid",
			prepData: func(t *testing.T) data.CreateConversation {
				return data.CreateConversation{
					Participants: []data.AddParticipant{
						{ParticipantID: "1234567890", Metadata: map[string]any{"business_id": "0987654321"}},
						{ParticipantID: "1111111111"},
					},
					Metadata: map[string]any{"hello": "world"},
				}
			},
			asserts: func(t *testing.T, conv *model.Conversation, err error) {
				assert.NoError(t, err)
				assert.NotEmpty(t, conv.ID.Hex())
				assert.False(t, conv.ID.IsZero())
				assert.Len(t, conv.Participants, 2)
				assert.Equal(t, "1234567890", conv.Participants[0].ParticipantID)
				assert.Equal(t, "1111111111", conv.Participants[1].ParticipantID)
				assert.Equal(t, "0987654321", conv.Participants[0].Metadata["business_id"])
				assert.Equal(t, "world", conv.Metadata["hello"])
			},
		},
		{
			name: "invalid with one participant",
			prepData: func(t *testing.T) data.CreateConversation {
				return data.CreateConversation{
					Participants: []data.AddParticipant{
						{ParticipantID: "1234567890", Metadata: map[string]any{"business_id": "0987654321"}},
					},
					Metadata: map[string]any{"hello": "world"},
				}
			},
			asserts: func(t *testing.T, conv *model.Conversation, err error) {
				assert.Error(t, err)
				assert.Nil(t, conv)
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			conv, err := cr.Create(t.Context(), tt.prepData(t))
			tt.asserts(t, conv, err)
		})
	}
}

func TestConversationRepository_Paginate(t *testing.T) {
	client := testutil.MustConnectMongoDB(t, os.Getenv("MONGODB_URI"))
	t.Cleanup(func() {
		client.Disconnect(nil)
	})

	cr := repository.NewConversation(client.Database("chatsavvy"))
	conv1, err := cr.Create(t.Context(), data.CreateConversation{
		Participants: []data.AddParticipant{
			{ParticipantID: "1234567890", Metadata: map[string]any{"business_id": "0987654321"}},
			{ParticipantID: "1111111111"},
		},
	})
	assert.NoError(t, err)
	time.Sleep(1 * time.Millisecond)

	conv2, err := cr.Create(t.Context(), data.CreateConversation{
		Participants: []data.AddParticipant{
			{ParticipantID: "1234567890", Metadata: map[string]any{"business_id": "999999999"}},
			{ParticipantID: "2222222222"},
		},
	})
	assert.NoError(t, err)
	time.Sleep(1 * time.Millisecond)

	conv3, err := cr.Create(t.Context(), data.CreateConversation{
		Participants: []data.AddParticipant{
			{ParticipantID: "2222222222"},
			{ParticipantID: "3333333333"},
			{ParticipantID: "1234567890"},
		},
	})
	assert.NoError(t, err)
	time.Sleep(1 * time.Millisecond)

	testCases := []struct {
		name     string
		prepData func(t *testing.T) data.PaginateConversations
		asserts  func(t *testing.T, convs []model.Conversation, total uint, err error)
	}{
		{
			name: "valid",
			prepData: func(t *testing.T) data.PaginateConversations {
				return data.PaginateConversations{
					ParticipantID: "1234567890",
					Page:          1,
					PerPage:       10,
				}
			},
			asserts: func(t *testing.T, convs []model.Conversation, total uint, err error) {
				assert.NoError(t, err)
				assert.Len(t, convs, 3)
				assert.Equal(t, uint(3), total)
				assert.Equal(t, conv3.ID.Hex(), convs[0].ID.Hex())
				assert.Equal(t, conv2.ID.Hex(), convs[1].ID.Hex())
				assert.Equal(t, conv1.ID.Hex(), convs[2].ID.Hex())
			},
		},
		{
			name: "valid with pagination",
			prepData: func(t *testing.T) data.PaginateConversations {
				return data.PaginateConversations{
					ParticipantID: "1234567890",
					Page:          1,
					PerPage:       2,
				}
			},
			asserts: func(t *testing.T, convs []model.Conversation, total uint, err error) {
				assert.NoError(t, err)
				assert.Len(t, convs, 2)
				assert.Equal(t, uint(3), total)
				assert.Equal(t, conv3.ID.Hex(), convs[0].ID.Hex())
				assert.Equal(t, conv2.ID.Hex(), convs[1].ID.Hex())
			},
		},
		{
			name: "invalid with invalid participant id",
			prepData: func(t *testing.T) data.PaginateConversations {
				return data.PaginateConversations{
					ParticipantID: "",
					Page:          1,
					PerPage:       2,
				}
			},
			asserts: func(t *testing.T, convs []model.Conversation, total uint, err error) {
				assert.Error(t, err)
				assert.Nil(t, convs)
				assert.Zero(t, total)
			},
		},
		{
			name: "invalid with invalid page",
			prepData: func(t *testing.T) data.PaginateConversations {
				return data.PaginateConversations{
					ParticipantID: "1234567890",
					Page:          0,
					PerPage:       2,
				}
			},
			asserts: func(t *testing.T, convs []model.Conversation, total uint, err error) {
				assert.Error(t, err)
				assert.Nil(t, convs)
				assert.Zero(t, total)
			},
		},
		{
			name: "invalid with invalid per page",
			prepData: func(t *testing.T) data.PaginateConversations {
				return data.PaginateConversations{
					ParticipantID: "1234567890",
					Page:          1,
					PerPage:       0,
				}
			},
			asserts: func(t *testing.T, convs []model.Conversation, total uint, err error) {
				assert.Error(t, err)
				assert.Nil(t, convs)
				assert.Zero(t, total)
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			convs, total, err := cr.Paginate(t.Context(), tt.prepData(t))
			tt.asserts(t, convs, total, err)
		})
	}
}

func TestConversationRepository_Find(t *testing.T) {
	client := testutil.MustConnectMongoDB(t, os.Getenv("MONGODB_URI"))
	t.Cleanup(func() {
		client.Disconnect(nil)
	})

	cr := repository.NewConversation(client.Database("chatsavvy"))
	conv, err := cr.Create(t.Context(), data.CreateConversation{
		Participants: []data.AddParticipant{
			{ParticipantID: "1234567890", Metadata: map[string]any{"business_id": "0987654321"}},
			{ParticipantID: "1111111111"},
		},
	})
	assert.NoError(t, err)

	testCases := []struct {
		name    string
		id      string
		asserts func(t *testing.T, conv *model.Conversation, err error)
	}{
		{
			name: "valid",
			id:   conv.ID.Hex(),
			asserts: func(t *testing.T, conv *model.Conversation, err error) {
				assert.NoError(t, err)
				assert.Equal(t, conv.ID.Hex(), conv.ID.Hex())
				assert.Len(t, conv.Participants, 2)
				assert.Equal(t, "1234567890", conv.Participants[0].ParticipantID)
				assert.Equal(t, "1111111111", conv.Participants[1].ParticipantID)
				assert.Equal(t, "0987654321", conv.Participants[0].Metadata["business_id"])
			},
		},
		{
			name: "invalid with invalid id",
			id:   "",
			asserts: func(t *testing.T, conv *model.Conversation, err error) {
				assert.Error(t, err)
				assert.Nil(t, conv)
			},
		},
		{
			name: "invalid with not found id",
			id:   "5f0c5b7a1a3f4b0c1c8d3b6b",
			asserts: func(t *testing.T, conv *model.Conversation, err error) {
				assert.NoError(t, err)
				assert.Nil(t, conv)
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			conv, err := cr.Find(t.Context(), tt.id)
			tt.asserts(t, conv, err)
		})
	}
}

func TestConversationRepository_AddParticipant(t *testing.T) {
	testCases := []struct {
		name        string
		setup       func(t *testing.T, cr *repository.Conversation) (*model.Conversation, *model.Conversation)
		participant func(t *testing.T) data.AddParticipant
		expects     func(t *testing.T, resultConv *model.Conversation, expectedConv *model.Conversation, err error)
	}{
		{
			name: "returns existing conversation if participants already exist",
			setup: func(t *testing.T, cr *repository.Conversation) (*model.Conversation, *model.Conversation) {
				conv1, err := cr.Create(t.Context(), data.CreateConversation{
					Participants: []data.AddParticipant{
						{ParticipantID: "1234567890", Metadata: map[string]any{"business_id": "0987654321"}},
						{ParticipantID: "2222222222", Metadata: map[string]any{"business_id": "999999999"}},
						{ParticipantID: "1111111111"},
					},
				})
				assert.NoError(t, err)

				conv2, err := cr.Create(t.Context(), data.CreateConversation{
					Participants: []data.AddParticipant{
						{ParticipantID: "1234567890", Metadata: map[string]any{"business_id": "0987654321"}},
						{ParticipantID: "2222222222", Metadata: map[string]any{"business_id": "999999999"}},
					},
				})
				assert.NoError(t, err)

				return conv1, conv2
			},
			participant: func(t *testing.T) data.AddParticipant {
				return data.AddParticipant{
					ParticipantID: "1111111111",
				}
			},
			expects: func(t *testing.T, resultConv *model.Conversation, expectedConv *model.Conversation, err error) {
				assert.NoError(t, err)
				assert.Equal(t, expectedConv.ID.Hex(), resultConv.ID.Hex())
				assert.Len(t, resultConv.Participants, 3)
				assert.Equal(t, expectedConv.Participants[0].ParticipantID, resultConv.Participants[0].ParticipantID)
				assert.Equal(t, expectedConv.Participants[1].ParticipantID, resultConv.Participants[1].ParticipantID)
				assert.Equal(t, expectedConv.Participants[2].ParticipantID, resultConv.Participants[2].ParticipantID)
				assert.Equal(t, expectedConv.Participants[0].Metadata["business_id"], resultConv.Participants[0].Metadata["business_id"])
				assert.Equal(t, expectedConv.Participants[1].Metadata["business_id"], resultConv.Participants[1].Metadata["business_id"])
			},
		},
		{
			name: "adds participant to the conversation if participants do not exist",
			setup: func(t *testing.T, cr *repository.Conversation) (*model.Conversation, *model.Conversation) {
				conv, err := cr.Create(t.Context(), data.CreateConversation{
					Participants: []data.AddParticipant{
						{ParticipantID: "1234567890", Metadata: map[string]any{"business_id": "0987654321"}},
						{ParticipantID: "2222222222", Metadata: map[string]any{"business_id": "999999999"}},
					},
				})
				assert.NoError(t, err)

				return conv, conv
			},
			participant: func(t *testing.T) data.AddParticipant {
				return data.AddParticipant{
					ParticipantID: "1111111111",
				}
			},
			expects: func(t *testing.T, resultConv *model.Conversation, expectedConv *model.Conversation, err error) {
				assert.NoError(t, err)
				assert.Equal(t, expectedConv.ID.Hex(), resultConv.ID.Hex())
				assert.Len(t, resultConv.Participants, 3)
			},
		},
		{
			name: "returns existing conversation if participant exists with metadata",
			setup: func(t *testing.T, cr *repository.Conversation) (*model.Conversation, *model.Conversation) {
				conv1, err := cr.Create(t.Context(), data.CreateConversation{
					Participants: []data.AddParticipant{
						{ParticipantID: "1234567890", Metadata: map[string]any{"business_id": "0987654321"}},
						{ParticipantID: "2222222222", Metadata: map[string]any{"business_id": "999999999"}},
						{ParticipantID: "1111111111"},
					},
				})
				assert.NoError(t, err)

				conv2, err := cr.Create(t.Context(), data.CreateConversation{
					Participants: []data.AddParticipant{
						{ParticipantID: "1234567890", Metadata: map[string]any{"business_id": "0987654321"}},
						{ParticipantID: "1111111111"},
					},
				})
				assert.NoError(t, err)

				return conv1, conv2
			},
			participant: func(t *testing.T) data.AddParticipant {
				return data.AddParticipant{
					ParticipantID: "2222222222",
					Metadata:      map[string]any{"business_id": "999999999"},
				}
			},
			expects: func(t *testing.T, resultConv *model.Conversation, expectedConv *model.Conversation, err error) {
				assert.NoError(t, err)
				assert.Equal(t, expectedConv.ID.Hex(), resultConv.ID.Hex())
				assert.Len(t, resultConv.Participants, 3)
				assert.Equal(t, expectedConv.Participants[0].ParticipantID, resultConv.Participants[0].ParticipantID)
				assert.Equal(t, expectedConv.Participants[1].ParticipantID, resultConv.Participants[1].ParticipantID)
				assert.Equal(t, expectedConv.Participants[2].ParticipantID, resultConv.Participants[2].ParticipantID)
				assert.Equal(t, expectedConv.Participants[0].Metadata["business_id"], resultConv.Participants[0].Metadata["business_id"])
				assert.Equal(t, expectedConv.Participants[1].Metadata["business_id"], resultConv.Participants[1].Metadata["business_id"])
			},
		},
		{
			name: "returns new conversation if participant exists with different metadata",
			setup: func(t *testing.T, cr *repository.Conversation) (*model.Conversation, *model.Conversation) {
				conv1, err := cr.Create(t.Context(), data.CreateConversation{
					Participants: []data.AddParticipant{
						{ParticipantID: "1234567890", Metadata: map[string]any{"business_id": "0987654321"}},
						{ParticipantID: "2222222222", Metadata: map[string]any{"business_id": "999999999"}},
						{ParticipantID: "1111111111"},
					},
				})
				assert.NoError(t, err)

				conv2, err := cr.Create(t.Context(), data.CreateConversation{
					Participants: []data.AddParticipant{
						{ParticipantID: "1234567890", Metadata: map[string]any{"business_id": "0987654321"}},
						{ParticipantID: "1111111111"},
					},
				})
				assert.NoError(t, err)

				return conv1, conv2
			},
			participant: func(t *testing.T) data.AddParticipant {
				return data.AddParticipant{
					ParticipantID: "2222222222",
					Metadata:      map[string]any{"business_id": "1111111111"},
				}
			},
			expects: func(t *testing.T, resultConv *model.Conversation, expectedConv *model.Conversation, err error) {
				assert.NoError(t, err)
				assert.NotEqual(t, expectedConv.ID.Hex(), resultConv.ID.Hex())
				assert.Len(t, resultConv.Participants, 3)
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			client := testutil.MustConnectMongoDB(t, os.Getenv("MONGODB_URI"))
			t.Cleanup(func() {
				client.Disconnect(nil)
			})

			cr := repository.NewConversation(client.Database("chatsavvy"))
			expectedConv, currentConv := tt.setup(t, cr)

			resultConv, err := cr.AddParticipant(t.Context(), currentConv.ID.Hex(), tt.participant(t))
			assert.NoError(t, err)

			tt.expects(t, resultConv, expectedConv, err)
		})
	}
}

func TestConversationRepository_DeleteParticipant(t *testing.T) {
	testCases := []struct {
		name        string
		setup       func(t *testing.T, cr *repository.Conversation) *model.Conversation
		participant func(t *testing.T) data.DeleteParticipant
		expects     func(t *testing.T, resultConv *model.Conversation, err error)
	}{
		{
			name: "deletes participant from the conversation by participant id only",
			setup: func(t *testing.T, cr *repository.Conversation) *model.Conversation {
				conv1, err := cr.Create(t.Context(), data.CreateConversation{
					Participants: []data.AddParticipant{
						{ParticipantID: "1234567890", Metadata: map[string]any{"business_id": "0987654321"}},
						{ParticipantID: "2222222222", Metadata: map[string]any{"business_id": "999999999"}},
						{ParticipantID: "1111111111"},
					},
				})
				assert.NoError(t, err)

				return conv1
			},
			participant: func(t *testing.T) data.DeleteParticipant {
				return data.DeleteParticipant{
					ParticipantID: "1111111111",
				}
			},
			expects: func(t *testing.T, resultConv *model.Conversation, err error) {
				assert.NoError(t, err)
				assert.Len(t, resultConv.Participants, 3)
				assert.NotNil(t, resultConv.Participants[2].DeletedAt)
			},
		},
		{
			name: "does not do anything if participant does not exist in the conversation",
			setup: func(t *testing.T, cr *repository.Conversation) *model.Conversation {
				conv, err := cr.Create(t.Context(), data.CreateConversation{
					Participants: []data.AddParticipant{
						{ParticipantID: "1234567890", Metadata: map[string]any{"business_id": "0987654321"}},
						{ParticipantID: "2222222222", Metadata: map[string]any{"business_id": "999999999"}},
					},
				})
				assert.NoError(t, err)

				return conv
			},
			participant: func(t *testing.T) data.DeleteParticipant {
				return data.DeleteParticipant{
					ParticipantID: "1111111111",
				}
			},
			expects: func(t *testing.T, resultConv *model.Conversation, err error) {
				assert.NoError(t, err)
				assert.Len(t, resultConv.Participants, 2)
			},
		},
		{
			name: "deletes participant from the conversation by participant id and metadata",
			setup: func(t *testing.T, cr *repository.Conversation) *model.Conversation {
				conv1, err := cr.Create(t.Context(), data.CreateConversation{
					Participants: []data.AddParticipant{
						{ParticipantID: "1234567890", Metadata: map[string]any{"business_id": "0987654321"}},
						{ParticipantID: "2222222222", Metadata: map[string]any{"business_id": "999999999"}},
						{ParticipantID: "1111111111"},
					},
				})
				assert.NoError(t, err)

				return conv1
			},
			participant: func(t *testing.T) data.DeleteParticipant {
				return data.DeleteParticipant{
					ParticipantID: "2222222222",
					Metadata:      map[string]any{"business_id": "999999999"},
				}
			},
			expects: func(t *testing.T, resultConv *model.Conversation, err error) {
				assert.NoError(t, err)
				assert.Len(t, resultConv.Participants, 3)
				assert.NotNil(t, resultConv.Participants[1].DeletedAt)
			},
		},
		{
			name: "does not delete participant if metadata does not match",
			setup: func(t *testing.T, cr *repository.Conversation) *model.Conversation {
				conv1, err := cr.Create(t.Context(), data.CreateConversation{
					Participants: []data.AddParticipant{
						{ParticipantID: "1234567890", Metadata: map[string]any{"business_id": "0987654321"}},
						{ParticipantID: "2222222222", Metadata: map[string]any{"business_id": "999999999"}},
						{ParticipantID: "1111111111"},
					},
				})
				assert.NoError(t, err)

				return conv1
			},
			participant: func(t *testing.T) data.DeleteParticipant {
				return data.DeleteParticipant{
					ParticipantID: "2222222222",
					Metadata:      map[string]any{"business_id": "1111111111"},
				}
			},
			expects: func(t *testing.T, resultConv *model.Conversation, err error) {
				assert.NoError(t, err)
				assert.Len(t, resultConv.Participants, 3)
				assert.Nil(t, resultConv.Participants[1].DeletedAt)
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			client := testutil.MustConnectMongoDB(t, os.Getenv("MONGODB_URI"))
			t.Cleanup(func() {
				client.Disconnect(nil)
			})

			cr := repository.NewConversation(client.Database("chatsavvy"))
			conv := tt.setup(t, cr)

			resultConv, err := cr.DeleteParticipant(t.Context(), conv.ID.Hex(), tt.participant(t))
			assert.NoError(t, err)

			tt.expects(t, resultConv, err)
		})
	}
}

func TestConversationRepository_UpdateLastMessage(t *testing.T) {
	client := testutil.MustConnectMongoDB(t, os.Getenv("MONGODB_URI"))
	t.Cleanup(func() {
		client.Disconnect(nil)
	})

	cr := repository.NewConversation(client.Database("chatsavvy"))
	conv, err := cr.Create(t.Context(), data.CreateConversation{
		Participants: []data.AddParticipant{
			{ParticipantID: "1234567890", Metadata: map[string]any{"business_id": "0987654321"}},
			{ParticipantID: "1111111111"},
		},
	})
	assert.NoError(t, err)

	err = cr.UpdateLastMessage(t.Context(), conv.ID.Hex(), model.Message{
		ID:             bson.NewObjectID(),
		ConversationID: conv.ID,
		Sender:         model.MessageSender{ParticipantID: "1234567890", Metadata: map[string]any{"business_id": "0987654321"}},
		Kind:           "general",
		Content:        "Hello, World!",
		CreatedAt:      time.Now(),
	})
	assert.NoError(t, err)

	conv, err = cr.Find(t.Context(), conv.ID.Hex())
	assert.NoError(t, err)
	assert.NotNil(t, conv.LastMessage)
	assert.Equal(t, "Hello, World!", conv.LastMessage.Content)
	assert.Equal(t, "1234567890", conv.LastMessage.Sender.ParticipantID)
	assert.Equal(t, "0987654321", conv.LastMessage.Sender.Metadata["business_id"])
}

func TestConversationRepository_ParticipantExists(t *testing.T) {
	client := testutil.MustConnectMongoDB(t, os.Getenv("MONGODB_URI"))
	t.Cleanup(func() {
		client.Disconnect(nil)
	})

	cr := repository.NewConversation(client.Database("chatsavvy"))
	conv, err := cr.Create(t.Context(), data.CreateConversation{
		Participants: []data.AddParticipant{
			{ParticipantID: "1234567890", Metadata: map[string]any{"business_id": "0987654321"}},
			{ParticipantID: "1111111111"},
		},
	})
	assert.NoError(t, err)

	testCases := []struct {
		name     string
		prepData func(t *testing.T) data.ParticipantExists
		asserts  func(t *testing.T, exists bool, err error)
	}{
		{
			name: "valid with participant id only",
			prepData: func(t *testing.T) data.ParticipantExists {
				return data.ParticipantExists{
					ParticipantID: "1234567890",
				}
			},
			asserts: func(t *testing.T, exists bool, err error) {
				assert.NoError(t, err)
				assert.True(t, exists)
			},
		},
		{
			name: "valid with participant id and metadata",
			prepData: func(t *testing.T) data.ParticipantExists {
				return data.ParticipantExists{
					ParticipantID: "1234567890",
					Metadata:      map[string]any{"business_id": "0987654321"},
				}
			},
			asserts: func(t *testing.T, exists bool, err error) {
				assert.NoError(t, err)
				assert.True(t, exists)
			},
		},
		{
			name: "non-existent participant id",
			prepData: func(t *testing.T) data.ParticipantExists {
				return data.ParticipantExists{
					ParticipantID: "2222222222",
				}
			},
			asserts: func(t *testing.T, exists bool, err error) {
				assert.NoError(t, err)
				assert.False(t, exists)
			},
		},
		{
			name: "non-existent participant id with existent metadata",
			prepData: func(t *testing.T) data.ParticipantExists {
				return data.ParticipantExists{
					ParticipantID: "2222222222",
					Metadata:      map[string]any{"business_id": "0987654321"},
				}
			},
			asserts: func(t *testing.T, exists bool, err error) {
				assert.NoError(t, err)
				assert.False(t, exists)
			},
		},
		{
			name: "existent participant id with non-existent metadata",
			prepData: func(t *testing.T) data.ParticipantExists {
				return data.ParticipantExists{
					ParticipantID: "1234567890",
					Metadata:      map[string]any{"business_id": "1111111111"},
				}
			},
			asserts: func(t *testing.T, exists bool, err error) {
				assert.NoError(t, err)
				assert.False(t, exists)
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			exists, err := cr.ParticipantExists(t.Context(), conv.ID.Hex(), tt.prepData(t))
			tt.asserts(t, exists, err)
		})
	}
}
