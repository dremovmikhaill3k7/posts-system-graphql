package graph

import (
	"context"
	"errors"

	"posts_service/internal/auth"
)

func (r *Resolver) setAuthCookie(ctx context.Context, userID int) error {
	w, ok := auth.ResponseWriterFromContext(ctx)
	if !ok {
		return errors.New("response writer not available")
	}
	return r.jwt.SetAuthCookie(w, userID)
}

func (r *Resolver) clearAuthCookie(ctx context.Context) {
	if w, ok := auth.ResponseWriterFromContext(ctx); ok {
		r.jwt.ClearAuthCookie(w)
	}
}
