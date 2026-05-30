package loaders

import (
	"context"
	"net/http"

	"posts_service/internal/repository"
)

func Middleware(repo repository.Repository, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ld := New(repo)
		ctx := context.WithValue(r.Context(), ctxKey{}, ld)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
