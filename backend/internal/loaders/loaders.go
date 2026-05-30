package loaders

import (
	"context"
	"fmt"
	"time"

	"github.com/vikstrous/dataloadgen"

	"posts_service/internal/graph/model"
	"posts_service/internal/repository"
)

type ctxKey struct{}

type RootCommentsKey struct {
	PostID int
	Limit  int
	Offset int
}

type Loaders struct {
	User         *dataloadgen.Loader[int, *model.User]
	Post         *dataloadgen.Loader[int, *model.Post]
	RootComments *dataloadgen.Loader[RootCommentsKey, []*model.Comment]
}

func New(repo repository.Repository) *Loaders {
	wait := dataloadgen.WithWait(time.Millisecond)
	return &Loaders{
		User:         dataloadgen.NewLoader(batchUsers(repo), wait),
		Post:         dataloadgen.NewLoader(batchPosts(repo), wait),
		RootComments: dataloadgen.NewLoader(batchRootComments(repo), wait),
	}
}

func For(ctx context.Context) *Loaders {
	return ctx.Value(ctxKey{}).(*Loaders)
}

func GetUser(ctx context.Context, id int) (*model.User, error) {
	return For(ctx).User.Load(ctx, id)
}

func GetPost(ctx context.Context, id int) (*model.Post, error) {
	return For(ctx).Post.Load(ctx, id)
}

func GetRootComments(ctx context.Context, postID, limit, offset int) ([]*model.Comment, error) {
	return For(ctx).RootComments.Load(ctx, RootCommentsKey{PostID: postID, Limit: limit, Offset: offset})
}

func batchUsers(repo repository.Repository) func(context.Context, []int) ([]*model.User, []error) {
	return func(ctx context.Context, ids []int) ([]*model.User, []error) {
		users, err := repo.GetUsersByIDs(ctx, ids)
		if err != nil {
			return nil, errsAll(err, len(ids))
		}
		out := make([]*model.User, len(ids))
		errs := make([]error, len(ids))
		for i, id := range ids {
			u, ok := users[id]
			if !ok {
				errs[i] = repository.ErrNotFound
				continue
			}
			out[i] = u
		}
		return out, errs
	}
}

func batchPosts(repo repository.Repository) func(context.Context, []int) ([]*model.Post, []error) {
	return func(ctx context.Context, ids []int) ([]*model.Post, []error) {
		posts, err := repo.GetPostsByIDs(ctx, ids)
		if err != nil {
			return nil, errsAll(err, len(ids))
		}
		out := make([]*model.Post, len(ids))
		errs := make([]error, len(ids))
		for i, id := range ids {
			p, ok := posts[id]
			if !ok {
				errs[i] = repository.ErrNotFound
				continue
			}
			out[i] = p
		}
		return out, errs
	}
}

func batchRootComments(repo repository.Repository) func(context.Context, []RootCommentsKey) ([][]*model.Comment, []error) {
	return func(ctx context.Context, keys []RootCommentsKey) ([][]*model.Comment, []error) {
		out := make([][]*model.Comment, len(keys))
		errs := make([]error, len(keys))

		groups := make(map[string][]int)
		for i, k := range keys {
			gk := fmt.Sprintf("%d:%d", k.Limit, k.Offset)
			groups[gk] = append(groups[gk], i)
		}

		for _, indices := range groups {
			if len(indices) == 0 {
				continue
			}
			sample := keys[indices[0]]
			postIDs := make([]int, 0, len(indices))
			seen := make(map[int]struct{}, len(indices))
			for _, idx := range indices {
				pid := keys[idx].PostID
				if _, ok := seen[pid]; ok {
					continue
				}
				seen[pid] = struct{}{}
				postIDs = append(postIDs, pid)
			}

			byPost, err := repo.ListRootCommentsForPosts(ctx, postIDs, sample.Limit, sample.Offset)
			if err != nil {
				for _, idx := range indices {
					errs[idx] = err
				}
				continue
			}
			for _, idx := range indices {
				pid := keys[idx].PostID
				out[idx] = byPost[pid]
				if out[idx] == nil {
					out[idx] = []*model.Comment{}
				}
			}
		}
		return out, errs
	}
}

func errsAll(err error, n int) []error {
	errs := make([]error, n)
	for i := range errs {
		errs[i] = err
	}
	return errs
}
