package auth

import (
	"context"
	"net/http"
)

type contextKey string

const (
	userIDKey          contextKey = "userID"
	responseWriterKey  contextKey = "responseWriter"
)

func WithUserID(ctx context.Context, userID int) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

func UserIDFromContext(ctx context.Context) (int, bool) {
	id, ok := ctx.Value(userIDKey).(int)
	return id, ok && id > 0
}

func ResponseWriterFromContext(ctx context.Context) (http.ResponseWriter, bool) {
	w, ok := ctx.Value(responseWriterKey).(http.ResponseWriter)
	return w, ok
}

func Middleware(jwt *JWT) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), responseWriterKey, w)
			if userID, ok := jwt.UserIDFromRequest(r); ok {
				ctx = WithUserID(ctx, userID)
			}
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
