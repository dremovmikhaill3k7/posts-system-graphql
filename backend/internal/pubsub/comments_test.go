package pubsub

import (
	"testing"
	"time"

	"posts_service/internal/graph/model"
)

func TestPublishSubscribe(t *testing.T) {
	ps := NewCommentPubSub()
	ch, unsub := ps.Subscribe(1)
	defer unsub()

	c := &model.Comment{ID: 42, Text: "hi"}
	ps.Publish(1, c)

	select {
	case got := <-ch:
		if got.ID != 42 {
			t.Fatalf("expected id 42, got %d", got.ID)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for comment")
	}
}
