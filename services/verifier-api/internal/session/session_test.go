package session_test

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/leoemaxie/trustpass/services/shared"
	"github.com/leoemaxie/trustpass/services/verifier-api/internal/session"
)

func TestSessionLifecycleAndReplay(t *testing.T) {
	store := session.NewMemoryStore()
	claim := shared.ClaimRequest{
		SchemaName:    "NationalIDCredential",
		AttributeName: "dateOfBirth",
		Operator:      shared.OpBeforeDate,
		Value:         json.RawMessage(`"2008-09-13"`),
	}

	// 1. Create session with 2 second TTL
	sess, err := store.CreateSession(claim, 2*time.Second)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	if sess.SessionToken == "" {
		t.Fatal("expected non-empty session token")
	}

	// 2. Retrieve valid session
	retrieved, err := store.Get(sess.SessionToken)
	if err != nil {
		t.Fatalf("expected valid session, got: %v", err)
	}
	if retrieved.Consumed {
		t.Fatal("expected new session to not be consumed")
	}

	// 3. Mark consumed
	if err := store.MarkConsumed(sess.SessionToken); err != nil {
		t.Fatalf("failed to mark consumed: %v", err)
	}

	// 4. Replay attempt on consumed session
	_, err = store.Get(sess.SessionToken)
	if !errors.Is(err, session.ErrSessionReused) {
		t.Fatalf("expected ErrSessionReused, got: %v", err)
	}

	// 5. Expiration test
	sessExpired, err := store.CreateSession(claim, 10*time.Millisecond)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}
	time.Sleep(20 * time.Millisecond)

	_, err = store.Get(sessExpired.SessionToken)
	if !errors.Is(err, session.ErrSessionExpired) {
		t.Fatalf("expected ErrSessionExpired, got: %v", err)
	}
}
