package data

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateMessage_Validate(t *testing.T) {
	testCases := []struct {
		name    string
		msg     CreateMessage
		wantErr bool
	}{
		{
			name: "content only",
			msg: CreateMessage{
				Kind:    "general",
				Sender:  MessageSender{ParticipantID: "123"},
				Content: "Hello",
			},
			wantErr: false,
		},
		{
			name: "attachments only",
			msg: CreateMessage{
				Kind:        "general",
				Sender:      MessageSender{ParticipantID: "123"},
				Attachments: []CreateAttachment{{Kind: "file"}},
			},
			wantErr: false,
		},
		{
			name: "both content and attachments",
			msg: CreateMessage{
				Kind:        "general",
				Sender:      MessageSender{ParticipantID: "123"},
				Content:     "Hello",
				Attachments: []CreateAttachment{{Kind: "file"}},
			},
			wantErr: false,
		},
		{
			name: "neither content nor attachments",
			msg: CreateMessage{
				Kind:   "general",
				Sender: MessageSender{ParticipantID: "123"},
			},
			wantErr: true,
		},
		{
			name: "empty string content no attachments",
			msg: CreateMessage{
				Kind:    "general",
				Sender:  MessageSender{ParticipantID: "123"},
				Content: "",
			},
			wantErr: true,
		},
		{
			name: "10 attachments no content",
			msg: CreateMessage{
				Kind:   "general",
				Sender: MessageSender{ParticipantID: "123"},
				Attachments: []CreateAttachment{
					{Kind: "file"}, {Kind: "file"}, {Kind: "file"}, {Kind: "file"}, {Kind: "file"},
					{Kind: "file"}, {Kind: "file"}, {Kind: "file"}, {Kind: "file"}, {Kind: "file"},
				},
			},
			wantErr: false,
		},
		{
			name: "11 attachments exceeds max",
			msg: CreateMessage{
				Kind:   "general",
				Sender: MessageSender{ParticipantID: "123"},
				Attachments: []CreateAttachment{
					{Kind: "file"}, {Kind: "file"}, {Kind: "file"}, {Kind: "file"}, {Kind: "file"},
					{Kind: "file"}, {Kind: "file"}, {Kind: "file"}, {Kind: "file"}, {Kind: "file"},
					{Kind: "file"},
				},
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.msg.Validate()
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
