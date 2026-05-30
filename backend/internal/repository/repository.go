package repository

import (
	"context"
	"errors"
	"posts_service/internal/graph/model"
)

const MaxCommentLength = 2000

var (
	ErrNotFound          = errors.New("not found")
	ErrForbidden         = errors.New("forbidden")
	ErrCommentsDisabled  = errors.New("comments are disabled for this post")
	ErrCommentTooLong    = errors.New("comment text exceeds maximum length")
	ErrInvalidParent     = errors.New("invalid parent comment")
	ErrEmailTaken        = errors.New("email already registered")
	ErrInvalidCredentials = errors.New("invalid credentials")
)

type Repository interface {
	CreateUser(ctx context.Context, email, username, passwordHash string) (*model.User, error)
	GetUserByEmail(ctx context.Context, email string) (*model.User, string, error)
	GetUserByID(ctx context.Context, id int) (*model.User, error)
	GetUsersByIDs(ctx context.Context, ids []int) (map[int]*model.User, error)
	ListUsers(ctx context.Context, limit, offset int, search string) ([]*model.User, error)

	CreatePost(ctx context.Context, userID int, text string, canHaveComm bool) (*model.Post, error)
	GetPostByID(ctx context.Context, id int) (*model.Post, error)
	GetPostsByIDs(ctx context.Context, ids []int) (map[int]*model.Post, error)
	ListPosts(ctx context.Context, limit, offset int, search string) ([]*model.Post, error)
	UpdatePostCommentsAllowed(ctx context.Context, userID, postID int, canHaveComm bool) (*model.Post, error)
	ListPostsByUser(ctx context.Context, userID, limit, offset int) ([]*model.Post, error)

	CreateComment(ctx context.Context, userID, postID int, parentID *int, text string) (*model.Comment, error)
	ListRootComments(ctx context.Context, postID, limit, offset int) ([]*model.Comment, error)
	ListRootCommentsForPosts(ctx context.Context, postIDs []int, limit, offset int) (map[int][]*model.Comment, error)
	ListReplyComments(ctx context.Context, parentID, limit, offset int) ([]*model.Comment, error)
	CommentsHaveReplies(ctx context.Context, commentIDs []int) (map[int]bool, error)
}
