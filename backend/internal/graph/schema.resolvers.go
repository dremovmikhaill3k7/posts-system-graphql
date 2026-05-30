package graph

import (
	"context"
	"errors"

	"golang.org/x/crypto/bcrypt"

	"posts_service/internal/auth"
	"posts_service/internal/graph/model"
	"posts_service/internal/loaders"
	"posts_service/internal/repository"
)

func (r *commentResolver) User(ctx context.Context, obj *model.Comment) (*model.User, error) {
	if obj.User == nil {
		return nil, nil
	}
	return r.loadUser(ctx, obj.User.ID)
}

func (r *commentResolver) Post(ctx context.Context, obj *model.Comment) (*model.Post, error) {
	if obj.Post == nil {
		return nil, nil
	}
	return r.loadPost(ctx, obj.Post.ID)
}

func (r *commentResolver) HasReplies(ctx context.Context, obj *model.Comment) (bool, error) {
	return loaders.GetHasReplies(ctx, obj.ID)
}

func (r *mutationResolver) Login(ctx context.Context, input model.LoginInput) (*model.User, error) {
	u, hash, err := r.repo.GetUserByEmail(ctx, input.Email)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, mapError(repository.ErrInvalidCredentials)
		}
		return nil, mapError(err)
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(input.Password)) != nil {
		return nil, mapError(repository.ErrInvalidCredentials)
	}
	if err := r.setAuthCookie(ctx, u.ID); err != nil {
		return nil, mapError(err)
	}
	return u, nil
}

func (r *mutationResolver) Register(ctx context.Context, input model.RegisterInput) (*model.User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, mapError(err)
	}
	u, err := r.repo.CreateUser(ctx, input.Email, input.Username, string(hash))
	if err != nil {
		return nil, mapError(err)
	}
	if err := r.setAuthCookie(ctx, u.ID); err != nil {
		return nil, mapError(err)
	}
	return u, nil
}

func (r *mutationResolver) Logout(ctx context.Context) (bool, error) {
	r.clearAuthCookie(ctx)
	return true, nil
}

func (r *mutationResolver) CreatePost(ctx context.Context, input model.CreatePostInput) (*model.Post, error) {
	userID, ok := auth.UserIDFromContext(ctx)
	if !ok {
		return nil, mapError(repository.ErrForbidden)
	}
	canHaveComm := true
	if input.CanHaveComm != nil {
		canHaveComm = *input.CanHaveComm
	}
	post, err := r.repo.CreatePost(ctx, userID, input.Text, canHaveComm)
	if err != nil {
		return nil, mapError(err)
	}
	return post, nil
}

func (r *mutationResolver) CreateComment(ctx context.Context, input model.CreateCommentInput) (*model.Comment, error) {
	userID, ok := auth.UserIDFromContext(ctx)
	if !ok {
		return nil, mapError(repository.ErrForbidden)
	}
	comment, err := r.repo.CreateComment(ctx, userID, input.PostID, input.ParentID, input.Text)
	if err != nil {
		return nil, mapError(err)
	}
	r.pubsub.Publish(input.PostID, comment)
	return comment, nil
}

func (r *mutationResolver) UpdatePost(ctx context.Context, input model.UpdatePostInput) (*model.Post, error) {
	userID, ok := auth.UserIDFromContext(ctx)
	if !ok {
		return nil, mapError(repository.ErrForbidden)
	}
	post, err := r.repo.UpdatePostCommentsAllowed(ctx, userID, input.PostID, input.CanHaveComm)
	if err != nil {
		return nil, mapError(err)
	}
	return post, nil
}

func (r *postResolver) User(ctx context.Context, obj *model.Post) (*model.User, error) {
	if obj.User == nil {
		return nil, nil
	}
	return r.loadUser(ctx, obj.User.ID)
}

func (r *postResolver) Comments(ctx context.Context, obj *model.Post, limit *int, offset *int) ([]*model.Comment, error) {
	l, o := limitOffset(limit, offset)
	comments, err := loaders.GetRootComments(ctx, obj.ID, l, o)
	if err != nil {
		return nil, mapError(err)
	}
	return comments, nil
}

func (r *queryResolver) User(ctx context.Context, id int) (*model.User, error) {
	u, err := r.loadUser(ctx, id)
	if err != nil {
		return nil, mapError(err)
	}
	return u, nil
}

func (r *queryResolver) Users(ctx context.Context, limit *int, offset *int, search *string) ([]*model.User, error) {
	l, o := limitOffset(limit, offset)
	s := ""
	if search != nil {
		s = *search
	}
	users, err := r.repo.ListUsers(ctx, l, o, s)
	if err != nil {
		return nil, mapError(err)
	}
	return users, nil
}

func (r *queryResolver) Post(ctx context.Context, id int) (*model.Post, error) {
	post, err := r.loadPost(ctx, id)
	if err != nil {
		return nil, mapError(err)
	}
	return post, nil
}

func (r *queryResolver) Posts(ctx context.Context, limit *int, offset *int, search *string) ([]*model.Post, error) {
	l, o := limitOffset(limit, offset)
	s := ""
	if search != nil {
		s = *search
	}
	posts, err := r.repo.ListPosts(ctx, l, o, s)
	if err != nil {
		return nil, mapError(err)
	}
	return posts, nil
}

func (r *queryResolver) ReplyComments(ctx context.Context, commentID int, limit *int, offset *int) ([]*model.Comment, error) {
	l, o := limitOffset(limit, offset)
	comments, err := r.repo.ListReplyComments(ctx, commentID, l, o)
	if err != nil {
		return nil, mapError(err)
	}
	return comments, nil
}

func (r *subscriptionResolver) CommentAdded(ctx context.Context, postID int) (<-chan *model.Comment, error) {
	ch, unsubscribe := r.pubsub.Subscribe(postID)
	go func() {
		<-ctx.Done()
		unsubscribe()
	}()
	return ch, nil
}

func (r *userResolver) Posts(ctx context.Context, obj *model.User, limit *int, offset *int) ([]*model.Post, error) {
	l, o := limitOffset(limit, offset)
	posts, err := r.repo.ListPostsByUser(ctx, obj.ID, l, o)
	if err != nil {
		return nil, mapError(err)
	}
	return posts, nil
}

func (r *Resolver) Comment() CommentResolver { return &commentResolver{r} }

func (r *Resolver) Mutation() MutationResolver { return &mutationResolver{r} }

func (r *Resolver) Post() PostResolver { return &postResolver{r} }

func (r *Resolver) Query() QueryResolver { return &queryResolver{r} }

func (r *Resolver) Subscription() SubscriptionResolver { return &subscriptionResolver{r} }

func (r *Resolver) User() UserResolver { return &userResolver{r} }

type commentResolver struct{ *Resolver }
type mutationResolver struct{ *Resolver }
type postResolver struct{ *Resolver }
type queryResolver struct{ *Resolver }
type subscriptionResolver struct{ *Resolver }
type userResolver struct{ *Resolver }
