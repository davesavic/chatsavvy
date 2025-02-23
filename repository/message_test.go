package repository_test

import (
	"fmt"
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

func TestMessageRepository_Create(t *testing.T) {
	client := testutil.MustConnectMongoDB(t, os.Getenv("MONGODB_URI"))
	t.Cleanup(func() {
		client.Disconnect(nil)
	})

	cr := repository.NewConversation(client.Database("chatsavvy"))
	mr := repository.NewMessage(client.Database("chatsavvy"), cr)

	testCases := []struct {
		name    string
		data    func(t *testing.T) data.CreateMessage
		setup   func(t *testing.T) *model.Conversation
		expects func(t *testing.T, msg *model.Message)
	}{
		{
			name: "it creates a general message",
			data: func(t *testing.T) data.CreateMessage {
				return data.CreateMessage{
					Kind: "general",
					Sender: data.MessageSender{
						ParticipantID: "1234567890",
					},
					Content: "Hello, World!",
				}
			},
			setup: func(t *testing.T) *model.Conversation {
				conv, err := cr.Create(t.Context(), data.CreateConversation{
					Participants: []data.AddParticipant{
						{ParticipantID: "1234567890"},
						{ParticipantID: "0987654321"},
					},
				})
				assert.NoError(t, err)
				return conv
			},
			expects: func(t *testing.T, msg *model.Message) {
				assert.NotNil(t, msg)
				assert.Equal(t, "general", msg.Kind)
				assert.Equal(t, "Hello, World!", msg.Content)
				assert.Equal(t, "1234567890", msg.Sender.ParticipantID)
			},
		},
		{
			name: "it creates a system message",
			data: func(t *testing.T) data.CreateMessage {
				return data.CreateMessage{
					Kind: "system",
					Sender: data.MessageSender{
						ParticipantID: "0000000000",
					},
					Content: "Hello, World!",
				}
			},
			setup: func(t *testing.T) *model.Conversation {
				conv, err := cr.Create(t.Context(), data.CreateConversation{
					Participants: []data.AddParticipant{
						{ParticipantID: "1234567890"},
						{ParticipantID: "0987654321"},
					},
				})
				assert.NoError(t, err)
				return conv
			},
			expects: func(t *testing.T, msg *model.Message) {
				assert.NotNil(t, msg)
				assert.Equal(t, "system", msg.Kind)
				assert.Equal(t, "Hello, World!", msg.Content)
				assert.Equal(t, "0000000000", msg.Sender.ParticipantID)
			},
		},
		{
			name: "it updates the conversation last message",
			data: func(t *testing.T) data.CreateMessage {
				return data.CreateMessage{
					Kind: "general",
					Sender: data.MessageSender{
						ParticipantID: "1234567890",
					},
					Content: "Hello, World!",
				}
			},
			setup: func(t *testing.T) *model.Conversation {
				conv, err := cr.Create(t.Context(), data.CreateConversation{
					Participants: []data.AddParticipant{
						{ParticipantID: "1234567890"},
						{ParticipantID: "0987654321"},
					},
				})
				assert.NoError(t, err)
				return conv
			},
			expects: func(t *testing.T, msg *model.Message) {
				assert.NotNil(t, msg)

				conv, err := cr.Find(t.Context(), msg.ConversationID.Hex())
				assert.NoError(t, err)
				assert.NotNil(t, conv)
				assert.Equal(t, msg.ID.Hex(), conv.LastMessage.ID.Hex())
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			conv := tc.setup(t)
			msg, err := mr.Create(t.Context(), conv.ID.Hex(), tc.data(t))
			assert.NoError(t, err)
			tc.expects(t, msg)
		})
	}
}

func TestMessageRepository_Paginate(t *testing.T) {
	testCases := []struct {
		name    string
		setup   func(t *testing.T, cr *repository.Conversation, mr *repository.Message) *model.Conversation
		data    func(t *testing.T, conv *model.Conversation) data.PaginateMessages
		expects func(t *testing.T, msgs []model.Message, total uint, err error)
	}{
		{
			name: "it paginates messages",
			setup: func(t *testing.T, cr *repository.Conversation, mr *repository.Message) *model.Conversation {
				conv, err := cr.Create(t.Context(), data.CreateConversation{
					Participants: []data.AddParticipant{
						{ParticipantID: "1234567890"},
						{ParticipantID: "0987654321"},
					},
				})
				assert.NoError(t, err)

				for i := 0; i < 3; i++ {
					_, err := mr.Create(t.Context(), conv.ID.Hex(), data.CreateMessage{
						Kind: "general",
						Sender: data.MessageSender{
							ParticipantID: "1234567890",
						},
						Content: fmt.Sprintf("Hello, World! %d", i),
					})
					assert.NoError(t, err)
					time.Sleep(1 * time.Millisecond)
				}

				return conv
			},
			data: func(t *testing.T, conv *model.Conversation) data.PaginateMessages {
				return data.PaginateMessages{
					ConversationID: conv.ID.Hex(),
					Page:           1,
					PerPage:        2,
				}
			},
			expects: func(t *testing.T, msgs []model.Message, total uint, err error) {
				assert.NoError(t, err)
				assert.Len(t, msgs, 2)
				assert.Equal(t, uint(3), total)
				assert.Equal(t, "Hello, World! 2", msgs[0].Content)
				assert.Equal(t, "Hello, World! 1", msgs[1].Content)
			},
		},
		{
			name: "it returns an empty list when there are no messages",
			setup: func(t *testing.T, cr *repository.Conversation, mr *repository.Message) *model.Conversation {
				conv, err := cr.Create(t.Context(), data.CreateConversation{
					Participants: []data.AddParticipant{
						{ParticipantID: "1234567890"},
						{ParticipantID: "0987654321"},
					},
				})
				assert.NoError(t, err)
				return conv
			},
			data: func(t *testing.T, conv *model.Conversation) data.PaginateMessages {
				return data.PaginateMessages{
					ConversationID: conv.ID.Hex(),
					Page:           1,
					PerPage:        2,
				}
			},
			expects: func(t *testing.T, msgs []model.Message, total uint, err error) {
				assert.NoError(t, err)
				assert.Len(t, msgs, 0)
				assert.Equal(t, uint(0), total)
			},
		},
		{
			name: "it returns an empty list when the conversation does not exist",
			setup: func(t *testing.T, cr *repository.Conversation, mr *repository.Message) *model.Conversation {
				return &model.Conversation{ID: bson.NewObjectID()}
			},
			data: func(t *testing.T, conv *model.Conversation) data.PaginateMessages {
				return data.PaginateMessages{
					ConversationID: conv.ID.Hex(),
					Page:           1,
					PerPage:        2,
				}
			},
			expects: func(t *testing.T, msgs []model.Message, total uint, err error) {
				assert.Error(t, err)
				assert.Nil(t, msgs)
				assert.Equal(t, uint(0), total)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client := testutil.MustConnectMongoDB(t, os.Getenv("MONGODB_URI"))
			t.Cleanup(func() {
				client.Disconnect(nil)
			})

			cr := repository.NewConversation(client.Database("chatsavvy"))
			mr := repository.NewMessage(client.Database("chatsavvy"), cr)

			conv := tc.setup(t, cr, mr)
			msgs, total, err := mr.Paginate(t.Context(), tc.data(t, conv))
			tc.expects(t, msgs, total, err)
		})
	}
}
