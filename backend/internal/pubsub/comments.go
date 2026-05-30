package pubsub

import (
	"sync"

	"posts_service/internal/graph/model"
)

type CommentPubSub struct {
	mu   sync.RWMutex
	subs map[int]map[chan *model.Comment]struct{}
}

func NewCommentPubSub() *CommentPubSub {
	return &CommentPubSub{
		subs: make(map[int]map[chan *model.Comment]struct{}),
	}
}

func (p *CommentPubSub) Subscribe(postID int) (<-chan *model.Comment, func()) {
	ch := make(chan *model.Comment, 8)

	p.mu.Lock()
	if p.subs[postID] == nil {
		p.subs[postID] = make(map[chan *model.Comment]struct{})
	}
	p.subs[postID][ch] = struct{}{}
	p.mu.Unlock()

	unsubscribe := func() {
		p.mu.Lock()
		defer p.mu.Unlock()
		if subs, ok := p.subs[postID]; ok {
			delete(subs, ch)
			if len(subs) == 0 {
				delete(p.subs, postID)
			}
		}
		close(ch)
	}

	return ch, unsubscribe
}

func (p *CommentPubSub) Publish(postID int, comment *model.Comment) {
	p.mu.RLock()
	subs := p.subs[postID]
	channels := make([]chan *model.Comment, 0, len(subs))
	for ch := range subs {
		channels = append(channels, ch)
	}
	p.mu.RUnlock()

	for _, ch := range channels {
		select {
		case ch <- comment:
		default:
		}
	}
}
