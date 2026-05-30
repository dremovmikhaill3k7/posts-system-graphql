package graph

import (
	"context"

	"posts_service/internal/graph/model"
	"posts_service/internal/loaders"
)

func (r *Resolver) loadUser(ctx context.Context, id int) (*model.User, error) {
	return loaders.GetUser(ctx, id)
}

func (r *Resolver) loadPost(ctx context.Context, id int) (*model.Post, error) {
	return loaders.GetPost(ctx, id)
}
