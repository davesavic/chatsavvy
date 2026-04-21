package repository_test

import (
	"fmt"
	"math/rand"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/davesavic/chatsavvy/data"
	"github.com/davesavic/chatsavvy/model"
	"github.com/davesavic/chatsavvy/repository"
	"github.com/davesavic/chatsavvy/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// findParticipant returns a pointer to the participant with the given ID (and metadata,
// if provided) in the conversation, or nil if not found. Matches the same exact-equality
// semantics used by the repository.
func findParticipant(conv *model.Conversation, participantID string, metadata map[string]any) *model.Participant {
	for i, p := range conv.Participants {
		if p.ParticipantID != participantID {
			continue
		}
		if metadata == nil && p.Metadata == nil {
			return &conv.Participants[i]
		}
		if len(p.Metadata) != len(metadata) {
			continue
		}
		match := true
		for k, v := range metadata {
			pv, ok := p.Metadata[k]
			if !ok || pv != v {
				match = false
				break
			}
		}
		if match {
			return &conv.Participants[i]
		}
	}
	return nil
}

func TestMessageRepository_MarkRead(t *testing.T) {
	client := testutil.MustConnectMongoDB(t, os.Getenv("MONGODB_URI"))
	t.Cleanup(func() { _ = client.Disconnect(t.Context()) })

	cr := repository.NewConversation(client.Database("chatsavvy"))
	mr := repository.NewMessage(client.Database("chatsavvy"), cr)

	// createConvWith3Messages creates a fresh conversation with two participants and 3 messages.
	// A unique pair of participant IDs is required to bypass Create's dedup-by-participants,
	// otherwise subtests accidentally share state via the same conversation document.
	createConvWith3Messages := func(t *testing.T, userA, userB string) (*model.Conversation, *model.Message, *model.Message, *model.Message) {
		t.Helper()
		conv, err := cr.Create(t.Context(), data.CreateConversation{
			Participants: []data.AddParticipant{
				{ParticipantID: userA},
				{ParticipantID: userB},
			},
		})
		require.NoError(t, err)

		m1, err := mr.Create(t.Context(), conv.ID.Hex(), data.CreateMessage{
			Kind:    "general",
			Sender:  data.MessageSender{ParticipantID: userA},
			Content: "m1",
		})
		require.NoError(t, err)

		m2, err := mr.Create(t.Context(), conv.ID.Hex(), data.CreateMessage{
			Kind:    "general",
			Sender:  data.MessageSender{ParticipantID: userA},
			Content: "m2",
		})
		require.NoError(t, err)

		m3, err := mr.Create(t.Context(), conv.ID.Hex(), data.CreateMessage{
			Kind:    "general",
			Sender:  data.MessageSender{ParticipantID: userA},
			Content: "m3",
		})
		require.NoError(t, err)

		return conv, m1, m2, m3
	}

	t.Run("advances cursor forward from nil", func(t *testing.T) {
		conv, _, m2, _ := createConvWith3Messages(t, "mr-fwd-a", "mr-fwd-b")

		updated, err := mr.MarkRead(t.Context(), data.MarkRead{
			ConversationID: conv.ID.Hex(),
			Participant:    data.ReadParticipant{ParticipantID: "mr-fwd-a"},
			MessageID:      m2.ID.Hex(),
		})
		require.NoError(t, err)
		require.NotNil(t, updated)

		p := findParticipant(updated, "mr-fwd-a", nil)
		require.NotNil(t, p)
		require.NotNil(t, p.LastReadMessageID)
		assert.Equal(t, m2.ID.Hex(), p.LastReadMessageID.Hex())
		require.NotNil(t, p.LastReadAt)
		// Allow tiny drift caused by Mongo's BSON datetime millisecond truncation.
		assert.WithinDuration(t, m2.CreatedAt, *p.LastReadAt, time.Millisecond)
	})

	t.Run("backward call is a no-op", func(t *testing.T) {
		conv, m1, m2, _ := createConvWith3Messages(t, "mr-bwd-a", "mr-bwd-b")

		_, err := mr.MarkRead(t.Context(), data.MarkRead{
			ConversationID: conv.ID.Hex(),
			Participant:    data.ReadParticipant{ParticipantID: "mr-bwd-a"},
			MessageID:      m2.ID.Hex(),
		})
		require.NoError(t, err)

		updated, err := mr.MarkRead(t.Context(), data.MarkRead{
			ConversationID: conv.ID.Hex(),
			Participant:    data.ReadParticipant{ParticipantID: "mr-bwd-a"},
			MessageID:      m1.ID.Hex(),
		})
		require.NoError(t, err)

		p := findParticipant(updated, "mr-bwd-a", nil)
		require.NotNil(t, p)
		require.NotNil(t, p.LastReadMessageID)
		assert.Equal(t, m2.ID.Hex(), p.LastReadMessageID.Hex())
	})

	t.Run("equal call is a no-op", func(t *testing.T) {
		conv, _, m2, _ := createConvWith3Messages(t, "mr-eq-a", "mr-eq-b")

		_, err := mr.MarkRead(t.Context(), data.MarkRead{
			ConversationID: conv.ID.Hex(),
			Participant:    data.ReadParticipant{ParticipantID: "mr-eq-a"},
			MessageID:      m2.ID.Hex(),
		})
		require.NoError(t, err)

		updated, err := mr.MarkRead(t.Context(), data.MarkRead{
			ConversationID: conv.ID.Hex(),
			Participant:    data.ReadParticipant{ParticipantID: "mr-eq-a"},
			MessageID:      m2.ID.Hex(),
		})
		require.NoError(t, err)

		p := findParticipant(updated, "mr-eq-a", nil)
		require.NotNil(t, p)
		require.NotNil(t, p.LastReadMessageID)
		assert.Equal(t, m2.ID.Hex(), p.LastReadMessageID.Hex())
	})

	t.Run("errors on cross-conversation message", func(t *testing.T) {
		convA, err := cr.Create(t.Context(), data.CreateConversation{
			Participants: []data.AddParticipant{
				{ParticipantID: "mr-cross-a1"},
				{ParticipantID: "mr-cross-a2"},
			},
		})
		require.NoError(t, err)

		convB, err := cr.Create(t.Context(), data.CreateConversation{
			Participants: []data.AddParticipant{
				{ParticipantID: "mr-cross-b1"},
				{ParticipantID: "mr-cross-b2"},
			},
		})
		require.NoError(t, err)

		// Message created in conv B.
		msgB, err := mr.Create(t.Context(), convB.ID.Hex(), data.CreateMessage{
			Kind:    "general",
			Sender:  data.MessageSender{ParticipantID: "mr-cross-b1"},
			Content: "hi b",
		})
		require.NoError(t, err)

		updated, err := mr.MarkRead(t.Context(), data.MarkRead{
			ConversationID: convA.ID.Hex(),
			Participant:    data.ReadParticipant{ParticipantID: "mr-cross-a1"},
			MessageID:      msgB.ID.Hex(),
		})
		require.Error(t, err)
		assert.Nil(t, updated)
		assert.Contains(t, err.Error(), "message does not belong to conversation")
	})

	t.Run("errors on absent participant", func(t *testing.T) {
		conv, _, m2, _ := createConvWith3Messages(t, "mr-absent-a", "mr-absent-b")

		updated, err := mr.MarkRead(t.Context(), data.MarkRead{
			ConversationID: conv.ID.Hex(),
			Participant:    data.ReadParticipant{ParticipantID: "mr-absent-ghost"},
			MessageID:      m2.ID.Hex(),
		})
		require.Error(t, err)
		assert.Nil(t, updated)
		assert.EqualError(t, err, "participant not found in conversation")
	})

	t.Run("errors on soft-deleted participant with identical error string", func(t *testing.T) {
		conv, _, m2, _ := createConvWith3Messages(t, "mr-soft-a", "mr-soft-b")

		_, err := cr.DeleteParticipant(t.Context(), conv.ID.Hex(), data.DeleteParticipant{
			ParticipantID: "mr-soft-a",
		})
		require.NoError(t, err)

		updated, err := mr.MarkRead(t.Context(), data.MarkRead{
			ConversationID: conv.ID.Hex(),
			Participant:    data.ReadParticipant{ParticipantID: "mr-soft-a"},
			MessageID:      m2.ID.Hex(),
		})
		require.Error(t, err)
		assert.Nil(t, updated)
		// Identical error string to the absent-participant case — no enumeration oracle.
		assert.EqualError(t, err, "participant not found in conversation")
	})

	t.Run("errors on invalid conversation id hex", func(t *testing.T) {
		conv, _, m2, _ := createConvWith3Messages(t, "mr-invhex-a", "mr-invhex-b")
		_ = conv

		updated, err := mr.MarkRead(t.Context(), data.MarkRead{
			ConversationID: "not-a-hex",
			Participant:    data.ReadParticipant{ParticipantID: "mr-invhex-a"},
			MessageID:      m2.ID.Hex(),
		})
		require.Error(t, err)
		assert.Nil(t, updated)
		assert.Contains(t, err.Error(), "failed to parse conversation id")
	})

	t.Run("non-nil metadata matches exactly and does not bleed across same-id participants", func(t *testing.T) {
		// Conversation with two participants sharing the same participant id, differing by metadata.
		conv, err := cr.Create(t.Context(), data.CreateConversation{
			Participants: []data.AddParticipant{
				{ParticipantID: "u1", Metadata: map[string]any{"org": "A"}},
				{ParticipantID: "u1", Metadata: map[string]any{"org": "B"}},
			},
		})
		require.NoError(t, err)

		m1, err := mr.Create(t.Context(), conv.ID.Hex(), data.CreateMessage{
			Kind:    "general",
			Sender:  data.MessageSender{ParticipantID: "u1", Metadata: map[string]any{"org": "A"}},
			Content: "m1",
		})
		require.NoError(t, err)

		updated, err := mr.MarkRead(t.Context(), data.MarkRead{
			ConversationID: conv.ID.Hex(),
			Participant:    data.ReadParticipant{ParticipantID: "u1", Metadata: map[string]any{"org": "A"}},
			MessageID:      m1.ID.Hex(),
		})
		require.NoError(t, err)

		pA := findParticipant(updated, "u1", map[string]any{"org": "A"})
		require.NotNil(t, pA)
		require.NotNil(t, pA.LastReadMessageID)
		assert.Equal(t, m1.ID.Hex(), pA.LastReadMessageID.Hex())

		pB := findParticipant(updated, "u1", map[string]any{"org": "B"})
		require.NotNil(t, pB)
		assert.Nil(t, pB.LastReadMessageID, "org=B participant's cursor should remain nil")
	})
}

func TestMessageRepository_MarkRead_UpdatedAtStable(t *testing.T) {
	client := testutil.MustConnectMongoDB(t, os.Getenv("MONGODB_URI"))
	t.Cleanup(func() { _ = client.Disconnect(t.Context()) })

	cr := repository.NewConversation(client.Database("chatsavvy"))
	mr := repository.NewMessage(client.Database("chatsavvy"), cr)

	// Fixture duplicated inline (see EPIC §Shared Context helper strategy): each subtest
	// uses a test-run-unique participant pair to bypass Conversation.Create's dedup.
	createConvWith3Messages := func(t *testing.T, userA, userB string) (*model.Conversation, *model.Message, *model.Message, *model.Message) {
		t.Helper()
		conv, err := cr.Create(t.Context(), data.CreateConversation{
			Participants: []data.AddParticipant{
				{ParticipantID: userA},
				{ParticipantID: userB},
			},
		})
		require.NoError(t, err)

		m1, err := mr.Create(t.Context(), conv.ID.Hex(), data.CreateMessage{
			Kind:    "general",
			Sender:  data.MessageSender{ParticipantID: userA},
			Content: "m1",
		})
		require.NoError(t, err)

		m2, err := mr.Create(t.Context(), conv.ID.Hex(), data.CreateMessage{
			Kind:    "general",
			Sender:  data.MessageSender{ParticipantID: userA},
			Content: "m2",
		})
		require.NoError(t, err)

		m3, err := mr.Create(t.Context(), conv.ID.Hex(), data.CreateMessage{
			Kind:    "general",
			Sender:  data.MessageSender{ParticipantID: userA},
			Content: "m3",
		})
		require.NoError(t, err)

		return conv, m1, m2, m3
	}

	t.Run("forward MarkRead does not bump updated_at", func(t *testing.T) {
		suffix := bson.NewObjectID().Hex()
		userA := "mr-ustab-fwd-a-" + suffix
		conv, _, m2, _ := createConvWith3Messages(t, userA, "mr-ustab-fwd-b-"+suffix)

		snap, err := cr.Find(t.Context(), conv.ID.Hex())
		require.NoError(t, err)
		require.NotNil(t, snap)
		before := snap.UpdatedAt

		_, err = mr.MarkRead(t.Context(), data.MarkRead{
			ConversationID: conv.ID.Hex(),
			Participant:    data.ReadParticipant{ParticipantID: userA},
			MessageID:      m2.ID.Hex(),
		})
		require.NoError(t, err)

		after, err := cr.Find(t.Context(), conv.ID.Hex())
		require.NoError(t, err)
		require.NotNil(t, after)
		// time.Time.Equal: BSON DateTime round-trip truncates to ms and may change the
		// internal monotonic reading, so byte/struct equality is the wrong operator.
		assert.True(t, before.Equal(after.UpdatedAt),
			"updated_at must not change on forward MarkRead (before=%s after=%s)",
			before, after.UpdatedAt)
	})

	t.Run("equal MarkRead does not bump updated_at", func(t *testing.T) {
		suffix := bson.NewObjectID().Hex()
		userA := "mr-ustab-eq-a-" + suffix
		conv, _, m2, _ := createConvWith3Messages(t, userA, "mr-ustab-eq-b-"+suffix)

		// Advance to M2 FIRST so the snapshot captures the post-setup updated_at; the
		// call-under-test is the second MarkRead(M2).
		_, err := mr.MarkRead(t.Context(), data.MarkRead{
			ConversationID: conv.ID.Hex(),
			Participant:    data.ReadParticipant{ParticipantID: userA},
			MessageID:      m2.ID.Hex(),
		})
		require.NoError(t, err)

		snap, err := cr.Find(t.Context(), conv.ID.Hex())
		require.NoError(t, err)
		require.NotNil(t, snap)
		before := snap.UpdatedAt

		_, err = mr.MarkRead(t.Context(), data.MarkRead{
			ConversationID: conv.ID.Hex(),
			Participant:    data.ReadParticipant{ParticipantID: userA},
			MessageID:      m2.ID.Hex(),
		})
		require.NoError(t, err)

		after, err := cr.Find(t.Context(), conv.ID.Hex())
		require.NoError(t, err)
		require.NotNil(t, after)
		assert.True(t, before.Equal(after.UpdatedAt),
			"updated_at must not change on equal MarkRead (before=%s after=%s)",
			before, after.UpdatedAt)
	})

	t.Run("backward MarkRead does not bump updated_at and preserves cursor", func(t *testing.T) {
		suffix := bson.NewObjectID().Hex()
		userA := "mr-ustab-bwd-a-" + suffix
		conv, m1, m2, _ := createConvWith3Messages(t, userA, "mr-ustab-bwd-b-"+suffix)

		_, err := mr.MarkRead(t.Context(), data.MarkRead{
			ConversationID: conv.ID.Hex(),
			Participant:    data.ReadParticipant{ParticipantID: userA},
			MessageID:      m2.ID.Hex(),
		})
		require.NoError(t, err)

		snap, err := cr.Find(t.Context(), conv.ID.Hex())
		require.NoError(t, err)
		require.NotNil(t, snap)
		before := snap.UpdatedAt

		returned, err := mr.MarkRead(t.Context(), data.MarkRead{
			ConversationID: conv.ID.Hex(),
			Participant:    data.ReadParticipant{ParticipantID: userA},
			MessageID:      m1.ID.Hex(),
		})
		require.NoError(t, err)
		require.NotNil(t, returned)

		// Return-contract: the ModifiedCount==0 disambiguation no-op path must hand back
		// a real conversation with the cursor preserved at M2 (not clobbered to nil or M1).
		p := findParticipant(returned, userA, nil)
		require.NotNil(t, p)
		require.NotNil(t, p.LastReadMessageID)
		assert.Equal(t, m2.ID.Hex(), p.LastReadMessageID.Hex())

		after, err := cr.Find(t.Context(), conv.ID.Hex())
		require.NoError(t, err)
		require.NotNil(t, after)
		assert.True(t, before.Equal(after.UpdatedAt),
			"updated_at must not change on backward MarkRead (before=%s after=%s)",
			before, after.UpdatedAt)
	})
}

func TestMessageRepository_MarkRead_PaginateOrderStable(t *testing.T) {
	client := testutil.MustConnectMongoDB(t, os.Getenv("MONGODB_URI"))
	t.Cleanup(func() { _ = client.Disconnect(t.Context()) })

	cr := repository.NewConversation(client.Database("chatsavvy"))
	mr := repository.NewMessage(client.Database("chatsavvy"), cr)

	suffix := bson.NewObjectID().Hex()
	sharedID := "paginate-order-user-" + suffix

	// Three conversations sharing only sharedID; distinct second participants prevent
	// Create's exact-participant-set dedup from collapsing them. 2ms sleeps between
	// creates keep BSON's ms-precision updated_at values distinct (Paginate's sort has
	// no secondary _id tie-break).
	createWithSeed := func(other string) *model.Conversation {
		conv, err := cr.Create(t.Context(), data.CreateConversation{
			Participants: []data.AddParticipant{
				{ParticipantID: sharedID},
				{ParticipantID: other},
			},
		})
		require.NoError(t, err)
		_, err = mr.Create(t.Context(), conv.ID.Hex(), data.CreateMessage{
			Kind:    "general",
			Sender:  data.MessageSender{ParticipantID: sharedID},
			Content: "seed",
		})
		require.NoError(t, err)
		return conv
	}

	c1 := createWithSeed("other-1-" + suffix)
	time.Sleep(2 * time.Millisecond)
	c2 := createWithSeed("other-2-" + suffix)
	time.Sleep(2 * time.Millisecond)
	c3 := createWithSeed("other-3-" + suffix)

	page := data.PaginateConversations{
		ParticipantID: sharedID,
		Page:          1,
		PerPage:       10,
	}

	collectIDs := func(t *testing.T) []bson.ObjectID {
		t.Helper()
		list, _, err := cr.Paginate(t.Context(), page)
		require.NoError(t, err)
		ids := make([]bson.ObjectID, 0, len(list))
		for _, c := range list {
			ids = append(ids, c.ID)
		}
		return ids
	}

	orderBefore := collectIDs(t)
	require.Equal(t, []bson.ObjectID{c3.ID, c2.ID, c1.ID}, orderBefore,
		"fixture precondition: Paginate returns newest-first by updated_at")

	c1Msgs, _, err := mr.Paginate(t.Context(), data.PaginateMessages{
		ConversationID: c1.ID.Hex(),
		Page:           1,
		PerPage:        10,
	})
	require.NoError(t, err)
	require.NotEmpty(t, c1Msgs)

	_, err = mr.MarkRead(t.Context(), data.MarkRead{
		ConversationID: c1.ID.Hex(),
		Participant:    data.ReadParticipant{ParticipantID: sharedID},
		MessageID:      c1Msgs[0].ID.Hex(),
	})
	require.NoError(t, err)

	orderAfter := collectIDs(t)
	assert.Equal(t, orderBefore, orderAfter,
		"Paginate order must not change after MarkRead — C1 must not jump to top")
}

func TestMessageRepository_MarkRead_PreflightSoftDelete(t *testing.T) {
	// Exercises the in-memory preflight fast-fail at repository/message.go:253-265.
	// The true preflight-passes-then-soft-delete TOCTOU (R1's post-update
	// ModifiedCount==0 disambiguation branch) is intentionally not covered at runtime;
	// see plan/read-receipts-remediation/EPIC.md §Non-Goals — its same-error-string
	// contract is enforced by code review against R1's AC.
	client := testutil.MustConnectMongoDB(t, os.Getenv("MONGODB_URI"))
	t.Cleanup(func() { _ = client.Disconnect(t.Context()) })

	cr := repository.NewConversation(client.Database("chatsavvy"))
	mr := repository.NewMessage(client.Database("chatsavvy"), cr)

	suffix := bson.NewObjectID().Hex()
	userA := "mr-pflsd-a-" + suffix
	userB := "mr-pflsd-b-" + suffix

	conv, err := cr.Create(t.Context(), data.CreateConversation{
		Participants: []data.AddParticipant{
			{ParticipantID: userA},
			{ParticipantID: userB},
		},
	})
	require.NoError(t, err)

	for i := 0; i < 3; i++ {
		_, err := mr.Create(t.Context(), conv.ID.Hex(), data.CreateMessage{
			Kind:    "general",
			Sender:  data.MessageSender{ParticipantID: userA},
			Content: fmt.Sprintf("m%d", i+1),
		})
		require.NoError(t, err)
	}

	// Re-fetch to get the final message set and pick a target.
	msgs, _, err := mr.Paginate(t.Context(), data.PaginateMessages{
		ConversationID: conv.ID.Hex(),
		Page:           1,
		PerPage:        10,
	})
	require.NoError(t, err)
	require.NotEmpty(t, msgs)
	targetMsgID := msgs[0].ID.Hex()

	_, err = cr.DeleteParticipant(t.Context(), conv.ID.Hex(), data.DeleteParticipant{
		ParticipantID: userA,
	})
	require.NoError(t, err)

	updated, err := mr.MarkRead(t.Context(), data.MarkRead{
		ConversationID: conv.ID.Hex(),
		Participant:    data.ReadParticipant{ParticipantID: userA},
		MessageID:      targetMsgID,
	})
	require.Error(t, err)
	assert.Nil(t, updated)
	assert.EqualError(t, err, "participant not found in conversation")
}

func TestMessageRepository_MarkAllRead(t *testing.T) {
	client := testutil.MustConnectMongoDB(t, os.Getenv("MONGODB_URI"))
	t.Cleanup(func() { _ = client.Disconnect(t.Context()) })

	cr := repository.NewConversation(client.Database("chatsavvy"))
	mr := repository.NewMessage(client.Database("chatsavvy"), cr)

	t.Run("resolves to true latest message", func(t *testing.T) {
		conv, err := cr.Create(t.Context(), data.CreateConversation{
			Participants: []data.AddParticipant{
				{ParticipantID: "mar-latest-alice"},
				{ParticipantID: "mar-latest-bob"},
			},
		})
		require.NoError(t, err)

		var m3 *model.Message
		for i := 0; i < 3; i++ {
			msg, err := mr.Create(t.Context(), conv.ID.Hex(), data.CreateMessage{
				Kind:    "general",
				Sender:  data.MessageSender{ParticipantID: "mar-latest-alice"},
				Content: fmt.Sprintf("m%d", i+1),
			})
			require.NoError(t, err)
			m3 = msg
		}

		updated, err := mr.MarkAllRead(t.Context(), data.MarkAllRead{
			ConversationID: conv.ID.Hex(),
			Participant:    data.ReadParticipant{ParticipantID: "mar-latest-alice"},
		})
		require.NoError(t, err)
		require.NotNil(t, updated)

		p := findParticipant(updated, "mar-latest-alice", nil)
		require.NotNil(t, p)
		require.NotNil(t, p.LastReadMessageID)
		assert.Equal(t, m3.ID.Hex(), p.LastReadMessageID.Hex())
	})

	t.Run("empty conversation is a no-op", func(t *testing.T) {
		// Unique participant IDs so Create's dedup doesn't hand us a conversation with messages
		// from a sibling subtest.
		conv, err := cr.Create(t.Context(), data.CreateConversation{
			Participants: []data.AddParticipant{
				{ParticipantID: "mar-empty-alice"},
				{ParticipantID: "mar-empty-bob"},
			},
		})
		require.NoError(t, err)

		updated, err := mr.MarkAllRead(t.Context(), data.MarkAllRead{
			ConversationID: conv.ID.Hex(),
			Participant:    data.ReadParticipant{ParticipantID: "mar-empty-alice"},
		})
		require.NoError(t, err)
		require.NotNil(t, updated)

		p := findParticipant(updated, "mar-empty-alice", nil)
		require.NotNil(t, p)
		assert.Nil(t, p.LastReadMessageID)
	})

	t.Run("idempotent across two calls", func(t *testing.T) {
		conv, err := cr.Create(t.Context(), data.CreateConversation{
			Participants: []data.AddParticipant{
				{ParticipantID: "mar-idem-alice"},
				{ParticipantID: "mar-idem-bob"},
			},
		})
		require.NoError(t, err)

		var m3 *model.Message
		for i := 0; i < 3; i++ {
			msg, err := mr.Create(t.Context(), conv.ID.Hex(), data.CreateMessage{
				Kind:    "general",
				Sender:  data.MessageSender{ParticipantID: "mar-idem-alice"},
				Content: fmt.Sprintf("m%d", i+1),
			})
			require.NoError(t, err)
			m3 = msg
		}

		_, err = mr.MarkAllRead(t.Context(), data.MarkAllRead{
			ConversationID: conv.ID.Hex(),
			Participant:    data.ReadParticipant{ParticipantID: "mar-idem-alice"},
		})
		require.NoError(t, err)

		updated, err := mr.MarkAllRead(t.Context(), data.MarkAllRead{
			ConversationID: conv.ID.Hex(),
			Participant:    data.ReadParticipant{ParticipantID: "mar-idem-alice"},
		})
		require.NoError(t, err)

		p := findParticipant(updated, "mar-idem-alice", nil)
		require.NotNil(t, p)
		require.NotNil(t, p.LastReadMessageID)
		assert.Equal(t, m3.ID.Hex(), p.LastReadMessageID.Hex())
	})
}

func TestMessageRepository_MarkRead_Concurrent(t *testing.T) {
	client := testutil.MustConnectMongoDB(t, os.Getenv("MONGODB_URI"))
	t.Cleanup(func() { _ = client.Disconnect(t.Context()) })

	cr := repository.NewConversation(client.Database("chatsavvy"))
	mr := repository.NewMessage(client.Database("chatsavvy"), cr)

	conv, err := cr.Create(t.Context(), data.CreateConversation{
		Participants: []data.AddParticipant{
			{ParticipantID: "alice"},
			{ParticipantID: "bob"},
		},
	})
	require.NoError(t, err)

	// Create 10 messages sequentially; byte-ordering of ObjectIDs matches creation order.
	msgs := make([]*model.Message, 10)
	for i := 0; i < 10; i++ {
		msg, err := mr.Create(t.Context(), conv.ID.Hex(), data.CreateMessage{
			Kind:    "general",
			Sender:  data.MessageSender{ParticipantID: "alice"},
			Content: fmt.Sprintf("m%d", i+1),
		})
		require.NoError(t, err)
		msgs[i] = msg
	}
	m10 := msgs[9]

	// Deterministic permutation for reproducibility.
	rng := rand.New(rand.NewSource(1))
	order := rng.Perm(len(msgs))

	ctx := t.Context()
	var wg sync.WaitGroup
	wg.Add(len(msgs))
	for _, idx := range order {
		idx := idx
		go func() {
			defer wg.Done()
			_, err := mr.MarkRead(ctx, data.MarkRead{
				ConversationID: conv.ID.Hex(),
				Participant:    data.ReadParticipant{ParticipantID: "alice"},
				MessageID:      msgs[idx].ID.Hex(),
			})
			// Each of these calls targets a valid message belonging to this conversation
			// for an existing participant — they must all succeed.
			assert.NoError(t, err)
		}()
	}
	wg.Wait()

	final, err := cr.Find(ctx, conv.ID.Hex())
	require.NoError(t, err)
	require.NotNil(t, final)

	p := findParticipant(final, "alice", nil)
	require.NotNil(t, p)
	require.NotNil(t, p.LastReadMessageID)
	assert.Equal(t, m10.ID.Hex(), p.LastReadMessageID.Hex(),
		"monotonicity: final cursor must be the byte-max message id (M10)")
}

func TestMessageRepository_ReadersOf(t *testing.T) {
	client := testutil.MustConnectMongoDB(t, os.Getenv("MONGODB_URI"))
	t.Cleanup(func() { _ = client.Disconnect(t.Context()) })

	cr := repository.NewConversation(client.Database("chatsavvy"))
	mr := repository.NewMessage(client.Database("chatsavvy"), cr)

	conv, err := cr.Create(t.Context(), data.CreateConversation{
		Participants: []data.AddParticipant{
			{ParticipantID: "A"},
			{ParticipantID: "B"},
			{ParticipantID: "C"},
			{ParticipantID: "D"},
		},
	})
	require.NoError(t, err)

	// Create 3 messages.
	m1, err := mr.Create(t.Context(), conv.ID.Hex(), data.CreateMessage{
		Kind:    "general",
		Sender:  data.MessageSender{ParticipantID: "A"},
		Content: "m1",
	})
	require.NoError(t, err)
	_ = m1

	m2, err := mr.Create(t.Context(), conv.ID.Hex(), data.CreateMessage{
		Kind:    "general",
		Sender:  data.MessageSender{ParticipantID: "A"},
		Content: "m2",
	})
	require.NoError(t, err)

	m3, err := mr.Create(t.Context(), conv.ID.Hex(), data.CreateMessage{
		Kind:    "general",
		Sender:  data.MessageSender{ParticipantID: "A"},
		Content: "m3",
	})
	require.NoError(t, err)

	// A → M3
	_, err = mr.MarkRead(t.Context(), data.MarkRead{
		ConversationID: conv.ID.Hex(),
		Participant:    data.ReadParticipant{ParticipantID: "A"},
		MessageID:      m3.ID.Hex(),
	})
	require.NoError(t, err)

	// B → M2
	_, err = mr.MarkRead(t.Context(), data.MarkRead{
		ConversationID: conv.ID.Hex(),
		Participant:    data.ReadParticipant{ParticipantID: "B"},
		MessageID:      m2.ID.Hex(),
	})
	require.NoError(t, err)

	// D → M3, then soft-delete D.
	_, err = mr.MarkRead(t.Context(), data.MarkRead{
		ConversationID: conv.ID.Hex(),
		Participant:    data.ReadParticipant{ParticipantID: "D"},
		MessageID:      m3.ID.Hex(),
	})
	require.NoError(t, err)
	_, err = cr.DeleteParticipant(t.Context(), conv.ID.Hex(), data.DeleteParticipant{
		ParticipantID: "D",
	})
	require.NoError(t, err)

	// C left with nil cursor.

	t.Run("ReadersOf(M2) returns {A, B}", func(t *testing.T) {
		readers, err := mr.ReadersOf(t.Context(), data.ReadersOf{
			ConversationID: conv.ID.Hex(),
			MessageID:      m2.ID.Hex(),
		})
		require.NoError(t, err)

		ids := make(map[string]struct{}, len(readers))
		for _, p := range readers {
			ids[p.ParticipantID] = struct{}{}
		}
		assert.Len(t, readers, 2)
		assert.Contains(t, ids, "A")
		assert.Contains(t, ids, "B")
	})

	t.Run("ReadersOf(M3) returns {A}", func(t *testing.T) {
		readers, err := mr.ReadersOf(t.Context(), data.ReadersOf{
			ConversationID: conv.ID.Hex(),
			MessageID:      m3.ID.Hex(),
		})
		require.NoError(t, err)

		ids := make(map[string]struct{}, len(readers))
		for _, p := range readers {
			ids[p.ParticipantID] = struct{}{}
		}
		assert.Len(t, readers, 1)
		assert.Contains(t, ids, "A")
	})

	t.Run("ReadersOf on non-existent conversation returns error", func(t *testing.T) {
		readers, err := mr.ReadersOf(t.Context(), data.ReadersOf{
			ConversationID: bson.NewObjectID().Hex(),
			MessageID:      m3.ID.Hex(),
		})
		require.Error(t, err)
		assert.Nil(t, readers)
	})
}

func TestMessageRepository_UnreadCount(t *testing.T) {
	client := testutil.MustConnectMongoDB(t, os.Getenv("MONGODB_URI"))
	t.Cleanup(func() { _ = client.Disconnect(t.Context()) })

	cr := repository.NewConversation(client.Database("chatsavvy"))
	mr := repository.NewMessage(client.Database("chatsavvy"), cr)

	t.Run("nil cursor returns total message count", func(t *testing.T) {
		conv, err := cr.Create(t.Context(), data.CreateConversation{
			Participants: []data.AddParticipant{
				{ParticipantID: "uc-nil-a"},
				{ParticipantID: "uc-nil-b"},
			},
		})
		require.NoError(t, err)

		for i := 0; i < 3; i++ {
			_, err := mr.Create(t.Context(), conv.ID.Hex(), data.CreateMessage{
				Kind:    "general",
				Sender:  data.MessageSender{ParticipantID: "uc-nil-b"},
				Content: fmt.Sprintf("m%d", i+1),
			})
			require.NoError(t, err)
		}

		count, err := mr.UnreadCount(t.Context(), data.UnreadCount{
			ConversationID: conv.ID.Hex(),
			Participant:    data.ReadParticipant{ParticipantID: "uc-nil-a"},
		})
		require.NoError(t, err)
		assert.Equal(t, uint(3), count)
	})

	t.Run("after MarkRead(latest) returns 0", func(t *testing.T) {
		conv, err := cr.Create(t.Context(), data.CreateConversation{
			Participants: []data.AddParticipant{
				{ParticipantID: "uc-latest-a"},
				{ParticipantID: "uc-latest-b"},
			},
		})
		require.NoError(t, err)

		var latest *model.Message
		for i := 0; i < 3; i++ {
			msg, err := mr.Create(t.Context(), conv.ID.Hex(), data.CreateMessage{
				Kind:    "general",
				Sender:  data.MessageSender{ParticipantID: "uc-latest-b"},
				Content: fmt.Sprintf("m%d", i+1),
			})
			require.NoError(t, err)
			latest = msg
		}

		_, err = mr.MarkRead(t.Context(), data.MarkRead{
			ConversationID: conv.ID.Hex(),
			Participant:    data.ReadParticipant{ParticipantID: "uc-latest-a"},
			MessageID:      latest.ID.Hex(),
		})
		require.NoError(t, err)

		count, err := mr.UnreadCount(t.Context(), data.UnreadCount{
			ConversationID: conv.ID.Hex(),
			Participant:    data.ReadParticipant{ParticipantID: "uc-latest-a"},
		})
		require.NoError(t, err)
		assert.Equal(t, uint(0), count)
	})

	t.Run("after MarkRead(M2) with 3 messages returns 1", func(t *testing.T) {
		conv, err := cr.Create(t.Context(), data.CreateConversation{
			Participants: []data.AddParticipant{
				{ParticipantID: "uc-m2-a"},
				{ParticipantID: "uc-m2-b"},
			},
		})
		require.NoError(t, err)

		msgs := make([]*model.Message, 3)
		for i := 0; i < 3; i++ {
			msg, err := mr.Create(t.Context(), conv.ID.Hex(), data.CreateMessage{
				Kind:    "general",
				Sender:  data.MessageSender{ParticipantID: "uc-m2-b"},
				Content: fmt.Sprintf("m%d", i+1),
			})
			require.NoError(t, err)
			msgs[i] = msg
		}

		_, err = mr.MarkRead(t.Context(), data.MarkRead{
			ConversationID: conv.ID.Hex(),
			Participant:    data.ReadParticipant{ParticipantID: "uc-m2-a"},
			MessageID:      msgs[1].ID.Hex(),
		})
		require.NoError(t, err)

		count, err := mr.UnreadCount(t.Context(), data.UnreadCount{
			ConversationID: conv.ID.Hex(),
			Participant:    data.ReadParticipant{ParticipantID: "uc-m2-a"},
		})
		require.NoError(t, err)
		assert.Equal(t, uint(1), count)
	})

	t.Run("mixed-kind messages are all counted", func(t *testing.T) {
		conv, err := cr.Create(t.Context(), data.CreateConversation{
			Participants: []data.AddParticipant{
				{ParticipantID: "uc-mix-a"},
				{ParticipantID: "uc-mix-b"},
			},
		})
		require.NoError(t, err)

		_, err = mr.Create(t.Context(), conv.ID.Hex(), data.CreateMessage{
			Kind:    "general",
			Sender:  data.MessageSender{ParticipantID: "uc-mix-b"},
			Content: "hello",
		})
		require.NoError(t, err)

		_, err = mr.Create(t.Context(), conv.ID.Hex(), data.CreateMessage{
			Kind:    "system",
			Sender:  data.MessageSender{ParticipantID: "uc-mix-system"},
			Content: "system notice",
		})
		require.NoError(t, err)

		count, err := mr.UnreadCount(t.Context(), data.UnreadCount{
			ConversationID: conv.ID.Hex(),
			Participant:    data.ReadParticipant{ParticipantID: "uc-mix-a"},
		})
		require.NoError(t, err)
		assert.Equal(t, uint(2), count)
	})

	t.Run("soft-deleted participant returns identical error string", func(t *testing.T) {
		conv, err := cr.Create(t.Context(), data.CreateConversation{
			Participants: []data.AddParticipant{
				{ParticipantID: "uc-soft-a"},
				{ParticipantID: "uc-soft-b"},
			},
		})
		require.NoError(t, err)

		_, err = mr.Create(t.Context(), conv.ID.Hex(), data.CreateMessage{
			Kind:    "general",
			Sender:  data.MessageSender{ParticipantID: "uc-soft-b"},
			Content: "m1",
		})
		require.NoError(t, err)

		_, err = cr.DeleteParticipant(t.Context(), conv.ID.Hex(), data.DeleteParticipant{
			ParticipantID: "uc-soft-a",
		})
		require.NoError(t, err)

		count, err := mr.UnreadCount(t.Context(), data.UnreadCount{
			ConversationID: conv.ID.Hex(),
			Participant:    data.ReadParticipant{ParticipantID: "uc-soft-a"},
		})
		require.Error(t, err)
		assert.Equal(t, uint(0), count)
		assert.EqualError(t, err, "participant not found in conversation")
	})

	t.Run("absent participant returns identical error string", func(t *testing.T) {
		conv, err := cr.Create(t.Context(), data.CreateConversation{
			Participants: []data.AddParticipant{
				{ParticipantID: "uc-absent-a"},
				{ParticipantID: "uc-absent-b"},
			},
		})
		require.NoError(t, err)

		_, err = mr.Create(t.Context(), conv.ID.Hex(), data.CreateMessage{
			Kind:    "general",
			Sender:  data.MessageSender{ParticipantID: "uc-absent-b"},
			Content: "m1",
		})
		require.NoError(t, err)

		count, err := mr.UnreadCount(t.Context(), data.UnreadCount{
			ConversationID: conv.ID.Hex(),
			Participant:    data.ReadParticipant{ParticipantID: "uc-absent-ghost"},
		})
		require.Error(t, err)
		assert.Equal(t, uint(0), count)
		assert.EqualError(t, err, "participant not found in conversation")
	})

	t.Run("own messages are not counted (nil cursor)", func(t *testing.T) {
		conv, err := cr.Create(t.Context(), data.CreateConversation{
			Participants: []data.AddParticipant{
				{ParticipantID: "uc-own-a"},
				{ParticipantID: "uc-own-b"},
			},
		})
		require.NoError(t, err)

		// Alice authors all three messages; Bob sends none.
		for i := 0; i < 3; i++ {
			_, err := mr.Create(t.Context(), conv.ID.Hex(), data.CreateMessage{
				Kind:    "general",
				Sender:  data.MessageSender{ParticipantID: "uc-own-a"},
				Content: fmt.Sprintf("m%d", i+1),
			})
			require.NoError(t, err)
		}

		count, err := mr.UnreadCount(t.Context(), data.UnreadCount{
			ConversationID: conv.ID.Hex(),
			Participant:    data.ReadParticipant{ParticipantID: "uc-own-a"},
		})
		require.NoError(t, err)
		assert.Equal(t, uint(0), count, "caller's own messages must not count as unread")
	})

	t.Run("mixed senders with nil cursor returns peer count only", func(t *testing.T) {
		conv, err := cr.Create(t.Context(), data.CreateConversation{
			Participants: []data.AddParticipant{
				{ParticipantID: "uc-mixed-a"},
				{ParticipantID: "uc-mixed-b"},
			},
		})
		require.NoError(t, err)

		_, err = mr.Create(t.Context(), conv.ID.Hex(), data.CreateMessage{
			Kind:    "general",
			Sender:  data.MessageSender{ParticipantID: "uc-mixed-a"},
			Content: "from-a",
		})
		require.NoError(t, err)
		for i := 0; i < 2; i++ {
			_, err := mr.Create(t.Context(), conv.ID.Hex(), data.CreateMessage{
				Kind:    "general",
				Sender:  data.MessageSender{ParticipantID: "uc-mixed-b"},
				Content: fmt.Sprintf("from-b-%d", i+1),
			})
			require.NoError(t, err)
		}

		count, err := mr.UnreadCount(t.Context(), data.UnreadCount{
			ConversationID: conv.ID.Hex(),
			Participant:    data.ReadParticipant{ParticipantID: "uc-mixed-a"},
		})
		require.NoError(t, err)
		assert.Equal(t, uint(2), count, "only peer messages should count as unread for Alice")
	})

	t.Run("own replies after MarkRead do not re-inflate count", func(t *testing.T) {
		conv, err := cr.Create(t.Context(), data.CreateConversation{
			Participants: []data.AddParticipant{
				{ParticipantID: "uc-reply-a"},
				{ParticipantID: "uc-reply-b"},
			},
		})
		require.NoError(t, err)

		peer, err := mr.Create(t.Context(), conv.ID.Hex(), data.CreateMessage{
			Kind:    "general",
			Sender:  data.MessageSender{ParticipantID: "uc-reply-b"},
			Content: "hi",
		})
		require.NoError(t, err)

		_, err = mr.MarkRead(t.Context(), data.MarkRead{
			ConversationID: conv.ID.Hex(),
			Participant:    data.ReadParticipant{ParticipantID: "uc-reply-a"},
			MessageID:      peer.ID.Hex(),
		})
		require.NoError(t, err)

		// Alice posts two replies after marking read.
		for i := 0; i < 2; i++ {
			_, err := mr.Create(t.Context(), conv.ID.Hex(), data.CreateMessage{
				Kind:    "general",
				Sender:  data.MessageSender{ParticipantID: "uc-reply-a"},
				Content: fmt.Sprintf("reply-%d", i+1),
			})
			require.NoError(t, err)
		}

		count, err := mr.UnreadCount(t.Context(), data.UnreadCount{
			ConversationID: conv.ID.Hex(),
			Participant:    data.ReadParticipant{ParticipantID: "uc-reply-a"},
		})
		require.NoError(t, err)
		assert.Equal(t, uint(0), count, "caller's own replies past their cursor must not count")
	})

	t.Run("metadata-sensitive sender identity", func(t *testing.T) {
		// Two participants sharing participant_id "u1" distinguished only by org metadata.
		conv, err := cr.Create(t.Context(), data.CreateConversation{
			Participants: []data.AddParticipant{
				{ParticipantID: "u1", Metadata: map[string]any{"org": "A"}},
				{ParticipantID: "u1", Metadata: map[string]any{"org": "B"}},
			},
		})
		require.NoError(t, err)

		// Message from (u1, org=A) — must NOT count for caller (u1, org=A).
		_, err = mr.Create(t.Context(), conv.ID.Hex(), data.CreateMessage{
			Kind:    "general",
			Sender:  data.MessageSender{ParticipantID: "u1", Metadata: map[string]any{"org": "A"}},
			Content: "from-A",
		})
		require.NoError(t, err)

		// Message from (u1, org=B) — MUST count for caller (u1, org=A).
		_, err = mr.Create(t.Context(), conv.ID.Hex(), data.CreateMessage{
			Kind:    "general",
			Sender:  data.MessageSender{ParticipantID: "u1", Metadata: map[string]any{"org": "B"}},
			Content: "from-B",
		})
		require.NoError(t, err)

		count, err := mr.UnreadCount(t.Context(), data.UnreadCount{
			ConversationID: conv.ID.Hex(),
			Participant:    data.ReadParticipant{ParticipantID: "u1", Metadata: map[string]any{"org": "A"}},
		})
		require.NoError(t, err)
		assert.Equal(t, uint(1), count, "only the (u1, org=B) message is unread for caller (u1, org=A)")
	})
}
