package graph

import (
	"errors"

	"github.com/vektah/gqlparser/v2/gqlerror"

	"posts_service/internal/repository"
)

func mapError(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, repository.ErrNotFound):
		return gqlerror.Errorf("resource not found")
	case errors.Is(err, repository.ErrForbidden):
		return gqlerror.Errorf("forbidden")
	case errors.Is(err, repository.ErrCommentsDisabled):
		return gqlerror.Errorf("comments are disabled for this post")
	case errors.Is(err, repository.ErrCommentTooLong):
		return gqlerror.Errorf("comment text must be at most %d characters", repository.MaxCommentLength)
	case errors.Is(err, repository.ErrInvalidParent):
		return gqlerror.Errorf("invalid parent comment")
	case errors.Is(err, repository.ErrEmailTaken):
		return gqlerror.Errorf("email already registered")
	case errors.Is(err, repository.ErrInvalidCredentials):
		return gqlerror.Errorf("invalid email or password")
	default:
		return gqlerror.Errorf("internal error: %v", err)
	}
}
