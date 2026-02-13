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
			name: "it creates a message with no attachments",
			data: func(t *testing.T) data.CreateMessage {
				return data.CreateMessage{
					Kind: "general",
					Sender: data.MessageSender{
						ParticipantID: "1234567890",
					},
					Content: "No attachments here",
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
				assert.Empty(t, msg.Attachments)
			},
		},
		{
			name: "it creates a message with a single attachment",
			data: func(t *testing.T) data.CreateMessage {
				return data.CreateMessage{
					Kind: "general",
					Sender: data.MessageSender{
						ParticipantID: "1234567890",
					},
					Content: "Check this file",
					Attachments: []data.CreateAttachment{
						{
							Kind: "file",
							Metadata: map[string]any{
								"filename":  "report.pdf",
								"size":      1024,
								"mime_type": "application/pdf",
								"url":       "https://example.com/report.pdf",
							},
						},
					},
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
				assert.Len(t, msg.Attachments, 1)
				assert.Equal(t, "file", msg.Attachments[0].Kind)
				assert.Equal(t, "report.pdf", msg.Attachments[0].Metadata["filename"])
			},
		},
		{
			name: "it creates a message with multiple attachments",
			data: func(t *testing.T) data.CreateMessage {
				return data.CreateMessage{
					Kind: "general",
					Sender: data.MessageSender{
						ParticipantID: "1234567890",
					},
					Content: "Here are some things",
					Attachments: []data.CreateAttachment{
						{
							Kind:     "file",
							Metadata: map[string]any{"filename": "photo.jpg"},
						},
						{
							Kind:     "link",
							Metadata: map[string]any{"url": "https://example.com"},
						},
						{
							Kind:     "location",
							Metadata: map[string]any{"lat": 40.7128, "lng": -74.0060},
						},
					},
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
				assert.Len(t, msg.Attachments, 3)
				assert.Equal(t, "file", msg.Attachments[0].Kind)
				assert.Equal(t, "link", msg.Attachments[1].Kind)
				assert.Equal(t, "location", msg.Attachments[2].Kind)
			},
		},
		{
			name: "it propagates attachments to conversation last_message",
			data: func(t *testing.T) data.CreateMessage {
				return data.CreateMessage{
					Kind: "general",
					Sender: data.MessageSender{
						ParticipantID: "1234567890",
					},
					Content: "With attachment",
					Attachments: []data.CreateAttachment{
						{
							Kind:     "file",
							Metadata: map[string]any{"filename": "doc.pdf"},
						},
					},
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
				assert.NotNil(t, conv.LastMessage)
				assert.Len(t, conv.LastMessage.Attachments, 1)
				assert.Equal(t, "file", conv.LastMessage.Attachments[0].Kind)
				assert.Equal(t, "doc.pdf", conv.LastMessage.Attachments[0].Metadata["filename"])
			},
		},
		{
			name: "it creates a message with attachments and no content",
			data: func(t *testing.T) data.CreateMessage {
				return data.CreateMessage{
					Kind: "general",
					Sender: data.MessageSender{
						ParticipantID: "1234567890",
					},
					Attachments: []data.CreateAttachment{
						{Kind: "file", Metadata: map[string]any{"filename": "photo.jpg"}},
						{Kind: "link", Metadata: map[string]any{"url": "https://example.com"}},
					},
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
				assert.Empty(t, msg.Content)
				assert.Len(t, msg.Attachments, 2)
			},
		},
		{
			name: "it propagates attachment-only message to conversation last_message",
			data: func(t *testing.T) data.CreateMessage {
				return data.CreateMessage{
					Kind: "general",
					Sender: data.MessageSender{
						ParticipantID: "1234567890",
					},
					Attachments: []data.CreateAttachment{
						{Kind: "file", Metadata: map[string]any{"filename": "doc.pdf"}},
					},
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
				assert.NotNil(t, conv.LastMessage)
				assert.Empty(t, conv.LastMessage.Content)
				assert.Len(t, conv.LastMessage.Attachments, 1)
				assert.Equal(t, "doc.pdf", conv.LastMessage.Attachments[0].Metadata["filename"])
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

	t.Run("it rejects a message with no content and no attachments", func(t *testing.T) {
		conv, err := cr.Create(t.Context(), data.CreateConversation{
			Participants: []data.AddParticipant{
				{ParticipantID: "1234567890"},
				{ParticipantID: "0987654321"},
			},
		})
		assert.NoError(t, err)

		msg, err := mr.Create(t.Context(), conv.ID.Hex(), data.CreateMessage{
			Kind:   "general",
			Sender: data.MessageSender{ParticipantID: "1234567890"},
		})
		assert.Error(t, err)
		assert.Nil(t, msg)
		assert.Contains(t, err.Error(), "content or at least one attachment")
	})
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

func TestMessageRepository_LoadMessages(t *testing.T) {
	testCases := []struct {
		name     string
		setup    func(t *testing.T, cr *repository.Conversation, mr *repository.Message) (*model.Conversation, *model.Message)
		loadData func(t *testing.T, conv *model.Conversation, lastMsg *model.Message) data.LoadMessages
		expects  func(t *testing.T, msgs []model.Message, err error)
	}{
		{
			name: "invalid load messages data",
			setup: func(t *testing.T, cr *repository.Conversation, mr *repository.Message) (*model.Conversation, *model.Message) {
				// No setup needed; we supply invalid data (e.g. empty conversation ID)
				return nil, nil
			},
			loadData: func(t *testing.T, conv *model.Conversation, lastMsg *model.Message) data.LoadMessages {
				return data.LoadMessages{
					ConversationID: "",
					LastMessageID:  (*string)(nil),
					PerPage:        10,
				}
			},
			expects: func(t *testing.T, msgs []model.Message, err error) {
				assert.Error(t, err)
			},
		},
		{
			name: "conversation not found",
			setup: func(t *testing.T, cr *repository.Conversation, mr *repository.Message) (*model.Conversation, *model.Message) {
				return nil, nil
			},
			loadData: func(t *testing.T, conv *model.Conversation, lastMsg *model.Message) data.LoadMessages {
				lmID := bson.NewObjectID().Hex()
				return data.LoadMessages{
					ConversationID: bson.NewObjectID().Hex(),
					LastMessageID:  &lmID,
					PerPage:        10,
				}
			},
			expects: func(t *testing.T, msgs []model.Message, err error) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "failed to fetch the conversation")
			},
		},
		{
			name: "invalid last message id",
			setup: func(t *testing.T, cr *repository.Conversation, mr *repository.Message) (*model.Conversation, *model.Message) {
				conv, err := cr.Create(t.Context(), data.CreateConversation{
					Participants: []data.AddParticipant{{ParticipantID: "123"}, {ParticipantID: "456"}},
				})
				assert.NoError(t, err)
				return conv, nil
			},
			loadData: func(t *testing.T, conv *model.Conversation, lastMsg *model.Message) data.LoadMessages {
				lmID := "invalid_hex"

				return data.LoadMessages{
					ConversationID: conv.ID.Hex(),
					LastMessageID:  &lmID,
					PerPage:        10,
				}
			},
			expects: func(t *testing.T, msgs []model.Message, err error) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "failed to parse the last message id")
			},
		},
		{
			name: "loads messages successfully",
			setup: func(t *testing.T, cr *repository.Conversation, mr *repository.Message) (*model.Conversation, *model.Message) {
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
						Content: fmt.Sprintf("Message %d", i),
					})
					assert.NoError(t, err)
					time.Sleep(1 * time.Millisecond)
				}
				lastMsg, err := mr.Create(t.Context(), conv.ID.Hex(), data.CreateMessage{
					Kind: "general",
					Sender: data.MessageSender{
						ParticipantID: "1234567890",
					},
					Content: "Latest message",
				})
				assert.NoError(t, err)
				return conv, lastMsg
			},
			loadData: func(t *testing.T, conv *model.Conversation, lastMsg *model.Message) data.LoadMessages {
				lmID := lastMsg.ID.Hex()

				return data.LoadMessages{
					ConversationID: conv.ID.Hex(),
					LastMessageID:  &lmID,
					PerPage:        2,
				}
			},
			expects: func(t *testing.T, msgs []model.Message, err error) {
				assert.NoError(t, err)
				assert.Len(t, msgs, 2)

				if len(msgs) == 2 {
					assert.True(t, msgs[0].ID.Hex() > msgs[1].ID.Hex(), "expected messages in descending order")
				}
			},
		},
		{
			name: "loads latest messages if last message id is nil",
			setup: func(t *testing.T, cr *repository.Conversation, mr *repository.Message) (*model.Conversation, *model.Message) {
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
						Content: fmt.Sprintf("Message %d", i),
					})
					assert.NoError(t, err)
					time.Sleep(1 * time.Millisecond)
				}
				lastMsg, err := mr.Create(t.Context(), conv.ID.Hex(), data.CreateMessage{
					Kind: "general",
					Sender: data.MessageSender{
						ParticipantID: "1234567890",
					},
					Content: "Latest message",
				})
				assert.NoError(t, err)
				return conv, lastMsg
			},
			loadData: func(t *testing.T, conv *model.Conversation, lastMsg *model.Message) data.LoadMessages {
				return data.LoadMessages{
					ConversationID: conv.ID.Hex(),
					LastMessageID:  nil,
					PerPage:        2,
				}
			},
			expects: func(t *testing.T, msgs []model.Message, err error) {
				assert.NoError(t, err)
				assert.Len(t, msgs, 2)

				if len(msgs) == 2 {
					assert.True(t, msgs[0].ID.Hex() > msgs[1].ID.Hex(), "expected messages in descending order")
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client := testutil.MustConnectMongoDB(t, os.Getenv("MONGODB_URI"))
			t.Cleanup(func() {
				client.Disconnect(t.Context())
			})

			cr := repository.NewConversation(client.Database("chatsavvy"))
			mr := repository.NewMessage(client.Database("chatsavvy"), cr)

			conv, lastMsg := tc.setup(t, cr, mr)
			loadData := tc.loadData(t, conv, lastMsg)
			msgs, err := mr.LoadMessages(t.Context(), loadData)
			tc.expects(t, msgs, err)
		})
	}
}

func TestMessageRepository_ToggleReaction(t *testing.T) {
	testCases := []struct {
		name    string
		setup   func(t *testing.T, cr *repository.Conversation, mr *repository.Message) (*model.Conversation, *model.Message)
		data    func(t *testing.T, conv *model.Conversation, msg *model.Message) data.ToggleReaction
		expects func(t *testing.T, msg *model.Message, err error)
	}{
		{
			name: "it sets a reaction on a message if it does not exist",
			setup: func(t *testing.T, cr *repository.Conversation, mr *repository.Message) (*model.Conversation, *model.Message) {
				conv, err := cr.Create(t.Context(), data.CreateConversation{
					Participants: []data.AddParticipant{{ParticipantID: "123"}, {ParticipantID: "456"}},
				})
				assert.NoError(t, err)

				msg, err := mr.Create(t.Context(), conv.ID.Hex(), data.CreateMessage{
					Kind: "general",
					Sender: data.MessageSender{
						ParticipantID: "123",
					},
					Content: "Hello, World!",
				})
				assert.NoError(t, err)

				return conv, msg
			},
			data: func(t *testing.T, conv *model.Conversation, msg *model.Message) data.ToggleReaction {
				return data.ToggleReaction{
					MessageID:   msg.ID.Hex(),
					Emoji:       ":thumbsup:",
					Participant: data.ReactionParticipant{ParticipantID: "123"},
				}
			},
			expects: func(t *testing.T, msg *model.Message, err error) {
				assert.NoError(t, err)
				assert.Len(t, msg.Reactions, 1)
				assert.Equal(t, ":thumbsup:", msg.Reactions[0].Emoji)
				assert.Len(t, msg.Reactions[0].Participants, 1)
				assert.Equal(t, "123", msg.Reactions[0].Participants[0].ParticipantID)
			},
		},
		{
			name: "it removes a reaction on a message if it exists",
			setup: func(t *testing.T, cr *repository.Conversation, mr *repository.Message) (*model.Conversation, *model.Message) {
				conv, err := cr.Create(t.Context(), data.CreateConversation{
					Participants: []data.AddParticipant{{ParticipantID: "123"}, {ParticipantID: "456"}},
				})
				assert.NoError(t, err)

				msg, err := mr.Create(t.Context(), conv.ID.Hex(), data.CreateMessage{
					Kind: "general",
					Sender: data.MessageSender{
						ParticipantID: "123",
					},
					Content: "Hello, World!",
				})
				assert.NoError(t, err)

				msg, err = mr.ToggleReaction(t.Context(), data.ToggleReaction{
					MessageID:   msg.ID.Hex(),
					Emoji:       ":thumbsup:",
					Participant: data.ReactionParticipant{ParticipantID: "123"},
				})
				assert.NoError(t, err)

				return conv, msg
			},
			data: func(t *testing.T, conv *model.Conversation, msg *model.Message) data.ToggleReaction {
				return data.ToggleReaction{
					MessageID:   msg.ID.Hex(),
					Emoji:       ":thumbsup:",
					Participant: data.ReactionParticipant{ParticipantID: "123"},
				}
			},
			expects: func(t *testing.T, msg *model.Message, err error) {
				assert.NoError(t, err)
				assert.Len(t, msg.Reactions, 0)
			},
		},
		{
			name: "message not found",
			setup: func(t *testing.T, cr *repository.Conversation, mr *repository.Message) (*model.Conversation, *model.Message) {
				conv, err := cr.Create(t.Context(), data.CreateConversation{
					Participants: []data.AddParticipant{{ParticipantID: "123"}, {ParticipantID: "456"}},
				})
				assert.NoError(t, err)
				return conv, nil
			},
			data: func(t *testing.T, conv *model.Conversation, msg *model.Message) data.ToggleReaction {
				return data.ToggleReaction{
					MessageID:   bson.NewObjectID().Hex(),
					Emoji:       ":thumbsup:",
					Participant: data.ReactionParticipant{ParticipantID: "123"},
				}
			},
			expects: func(t *testing.T, msg *model.Message, err error) {
				assert.Error(t, err)
				assert.Nil(t, msg)
				assert.Contains(t, err.Error(), "failed to fetch the message")
			},
		},
		{
			name: "adds a reaction to a message with existing same reaction",
			setup: func(t *testing.T, cr *repository.Conversation, mr *repository.Message) (*model.Conversation, *model.Message) {
				conv, err := cr.Create(t.Context(), data.CreateConversation{
					Participants: []data.AddParticipant{{ParticipantID: "123"}, {ParticipantID: "456"}},
				})
				assert.NoError(t, err)

				msg, err := mr.Create(t.Context(), conv.ID.Hex(), data.CreateMessage{
					Kind: "general",
					Sender: data.MessageSender{
						ParticipantID: "123",
					},
					Content: "Hello, World!",
				})
				assert.NoError(t, err)
				time.Sleep(1 * time.Millisecond)

				msg, err = mr.ToggleReaction(t.Context(), data.ToggleReaction{
					MessageID:   msg.ID.Hex(),
					Emoji:       ":thumbsup:",
					Participant: data.ReactionParticipant{ParticipantID: "456"},
				})
				assert.NoError(t, err)

				return conv, msg
			},
			data: func(t *testing.T, conv *model.Conversation, msg *model.Message) data.ToggleReaction {
				return data.ToggleReaction{
					MessageID:   msg.ID.Hex(),
					Emoji:       ":thumbsup:",
					Participant: data.ReactionParticipant{ParticipantID: "123"},
				}
			},
			expects: func(t *testing.T, msg *model.Message, err error) {
				assert.NoError(t, err)
				assert.Len(t, msg.Reactions, 1)
				assert.Equal(t, ":thumbsup:", msg.Reactions[0].Emoji)
				assert.Len(t, msg.Reactions[0].Participants, 2)
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

			conv, msg := tc.setup(t, cr, mr)
			data := tc.data(t, conv, msg)
			msg, err := mr.ToggleReaction(t.Context(), data)
			tc.expects(t, msg, err)
		})
	}
}
