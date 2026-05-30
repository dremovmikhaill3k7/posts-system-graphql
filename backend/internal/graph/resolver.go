package graph

import (
	"posts_service/internal/auth"
	"posts_service/internal/pubsub"
	"posts_service/internal/repository"
)

type Resolver struct {
	repo   repository.Repository
	pubsub *pubsub.CommentPubSub
	jwt    *auth.JWT
}

func NewResolver(repo repository.Repository, ps *pubsub.CommentPubSub, jwt *auth.JWT) *Resolver {
	return &Resolver{repo: repo, pubsub: ps, jwt: jwt}
}
