package memory

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"posts_service/internal/graph/model"
	"posts_service/internal/repository"
)

type Repository struct {
	mu        sync.RWMutex
	users     map[int]*userRecord
	usersByEmail map[string]int
	posts     map[int]*model.Post
	comments  map[int]*model.Comment
	nextUserID    int
	nextPostID    int
	nextCommentID int
}

type userRecord struct {
	user         *model.User
	passwordHash string
}

func NewRepository() *Repository {
	return &Repository{
		users:        make(map[int]*userRecord),
		usersByEmail: make(map[string]int),
		posts:        make(map[int]*model.Post),
		comments:     make(map[int]*model.Comment),
		nextUserID:   1,
		nextPostID:   1,
		nextCommentID: 1,
	}
}

func (r *Repository) CreateUser(_ context.Context, email, username, passwordHash string) (*model.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.usersByEmail[email]; ok {
		return nil, repository.ErrEmailTaken
	}

	now := time.Now()
	u := &model.User{
		ID:        r.nextUserID,
		Username:  username,
		Email:     email,
		Status:    model.TypeStatusActive,
		CreatedAt: &now,
	}
	r.users[u.ID] = &userRecord{user: cloneUser(u), passwordHash: passwordHash}
	r.usersByEmail[email] = u.ID
	r.nextUserID++
	return cloneUser(u), nil
}

func (r *Repository) GetUserByEmail(_ context.Context, email string) (*model.User, string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	id, ok := r.usersByEmail[email]
	if !ok {
		return nil, "", repository.ErrNotFound
	}
	rec := r.users[id]
	return cloneUser(rec.user), rec.passwordHash, nil
}

func (r *Repository) GetUserByID(ctx context.Context, id int) (*model.User, error) {
	users, err := r.GetUsersByIDs(ctx, []int{id})
	if err != nil {
		return nil, err
	}
	u, ok := users[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return u, nil
}

func (r *Repository) GetUsersByIDs(_ context.Context, ids []int) (map[int]*model.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[int]*model.User, len(ids))
	for _, id := range ids {
		if rec, ok := r.users[id]; ok {
			result[id] = cloneUser(rec.user)
		}
	}
	return result, nil
}

func (r *Repository) ListUsers(_ context.Context, limit, offset int, search string) ([]*model.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var list []*model.User
	for _, rec := range r.users {
		if rec.user.Status != model.TypeStatusActive {
			continue
		}
		if search != "" && !matchesSearch(rec.user.Username, rec.user.Email, search) {
			continue
		}
		list = append(list, cloneUser(rec.user))
	}
	sort.Slice(list, func(i, j int) bool { return list[i].ID < list[j].ID })
	return paginateUsers(list, limit, offset), nil
}

func (r *Repository) CreatePost(_ context.Context, userID int, text string, canHaveComm bool) (*model.Post, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.users[userID]; !ok {
		return nil, repository.ErrNotFound
	}

	now := time.Now()
	p := &model.Post{
		ID:          r.nextPostID,
		User:        &model.User{ID: userID},
		Text:        text,
		CanHaveComm: canHaveComm,
		Status:      model.TypeStatusActive,
		CreatedAt:   &now,
		UpdatedAt:   &now,
	}
	r.posts[p.ID] = clonePost(p)
	r.nextPostID++
	return clonePost(p), nil
}

func (r *Repository) GetPostsByIDs(_ context.Context, ids []int) (map[int]*model.Post, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[int]*model.Post, len(ids))
	for _, id := range ids {
		if p, ok := r.posts[id]; ok && p.Status == model.TypeStatusActive {
			result[id] = clonePost(p)
		}
	}
	return result, nil
}

func (r *Repository) GetPostByID(_ context.Context, id int) (*model.Post, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	p, ok := r.posts[id]
	if !ok || p.Status != model.TypeStatusActive {
		return nil, repository.ErrNotFound
	}
	return clonePost(p), nil
}

func (r *Repository) ListPosts(_ context.Context, limit, offset int, search string) ([]*model.Post, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var list []*model.Post
	for _, p := range r.posts {
		if p.Status != model.TypeStatusActive {
			continue
		}
		if search != "" && !strings.Contains(strings.ToLower(p.Text), strings.ToLower(search)) {
			continue
		}
		list = append(list, clonePost(p))
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].CreatedAt.After(*list[j].CreatedAt)
	})
	return paginatePosts(list, limit, offset), nil
}

func (r *Repository) ListPostsByUser(_ context.Context, userID, limit, offset int) ([]*model.Post, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var list []*model.Post
	for _, p := range r.posts {
		if p.User.ID != userID || p.Status != model.TypeStatusActive {
			continue
		}
		list = append(list, clonePost(p))
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].CreatedAt.After(*list[j].CreatedAt)
	})
	return paginatePosts(list, limit, offset), nil
}

func (r *Repository) UpdatePostCommentsAllowed(_ context.Context, userID, postID int, canHaveComm bool) (*model.Post, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	p, ok := r.posts[postID]
	if !ok || p.Status != model.TypeStatusActive {
		return nil, repository.ErrNotFound
	}
	if p.User.ID != userID {
		return nil, repository.ErrForbidden
	}
	p.CanHaveComm = canHaveComm
	now := time.Now()
	p.UpdatedAt = &now
	return clonePost(p), nil
}

func (r *Repository) CreateComment(_ context.Context, userID, postID int, parentID *int, text string) (*model.Comment, error) {
	if len([]rune(text)) > repository.MaxCommentLength {
		return nil, repository.ErrCommentTooLong
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	post, ok := r.posts[postID]
	if !ok || post.Status != model.TypeStatusActive {
		return nil, repository.ErrNotFound
	}
	if !post.CanHaveComm {
		return nil, repository.ErrCommentsDisabled
	}
	if _, ok := r.users[userID]; !ok {
		return nil, repository.ErrNotFound
	}

	if parentID != nil {
		parent, ok := r.comments[*parentID]
		if !ok || parent.Status != model.TypeStatusActive || parent.Post.ID != postID {
			return nil, repository.ErrInvalidParent
		}
	}

	now := time.Now()
	c := &model.Comment{
		ID:        r.nextCommentID,
		ParentID:  parentID,
		User:      &model.User{ID: userID},
		Post:      &model.Post{ID: postID},
		Text:      text,
		Status:    model.TypeStatusActive,
		CreatedAt: &now,
		UpdatedAt: &now,
	}
	r.comments[c.ID] = cloneComment(c)
	r.nextCommentID++
	return cloneComment(c), nil
}

func (r *Repository) ListRootCommentsForPosts(_ context.Context, postIDs []int, limit, offset int) (map[int][]*model.Comment, error) {
	result := make(map[int][]*model.Comment, len(postIDs))
	for _, id := range postIDs {
		list, err := r.ListRootComments(context.Background(), id, limit, offset)
		if err != nil {
			return nil, err
		}
		result[id] = list
	}
	return result, nil
}

func (r *Repository) ListRootComments(_ context.Context, postID, limit, offset int) ([]*model.Comment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var list []*model.Comment
	for _, c := range r.comments {
		if c.Post.ID != postID || c.ParentID != nil || c.Status != model.TypeStatusActive {
			continue
		}
		list = append(list, cloneComment(c))
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].CreatedAt.Before(*list[j].CreatedAt)
	})
	return paginateComments(list, limit, offset), nil
}

func (r *Repository) ListReplyComments(_ context.Context, parentID, limit, offset int) ([]*model.Comment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var list []*model.Comment
	for _, c := range r.comments {
		if c.ParentID == nil || *c.ParentID != parentID || c.Status != model.TypeStatusActive {
			continue
		}
		list = append(list, cloneComment(c))
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].CreatedAt.Before(*list[j].CreatedAt)
	})
	return paginateComments(list, limit, offset), nil
}

func (r *Repository) CommentsHaveReplies(_ context.Context, commentIDs []int) (map[int]bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[int]bool, len(commentIDs))
	for _, id := range commentIDs {
		result[id] = false
	}

	for _, c := range r.comments {
		if c.ParentID == nil || c.Status != model.TypeStatusActive {
			continue
		}
		if _, ok := result[*c.ParentID]; ok {
			result[*c.ParentID] = true
		}
	}
	return result, nil
}

func paginateUsers(list []*model.User, limit, offset int) []*model.User {
	if offset >= len(list) {
		return []*model.User{}
	}
	end := offset + limit
	if end > len(list) {
		end = len(list)
	}
	return list[offset:end]
}

func paginatePosts(list []*model.Post, limit, offset int) []*model.Post {
	if offset >= len(list) {
		return []*model.Post{}
	}
	end := offset + limit
	if end > len(list) {
		end = len(list)
	}
	return list[offset:end]
}

func paginateComments(list []*model.Comment, limit, offset int) []*model.Comment {
	if offset >= len(list) {
		return []*model.Comment{}
	}
	end := offset + limit
	if end > len(list) {
		end = len(list)
	}
	return list[offset:end]
}

func matchesSearch(username, email, search string) bool {
	s := strings.ToLower(search)
	return strings.Contains(strings.ToLower(username), s) ||
		strings.Contains(strings.ToLower(email), s)
}

func cloneUser(u *model.User) *model.User {
	cp := *u
	return &cp
}

func clonePost(p *model.Post) *model.Post {
	cp := *p
	if p.User != nil {
		u := *p.User
		cp.User = &u
	}
	return &cp
}

func cloneComment(c *model.Comment) *model.Comment {
	cp := *c
	if c.User != nil {
		u := *c.User
		cp.User = &u
	}
	if c.Post != nil {
		p := *c.Post
		cp.Post = &p
	}
	if c.ParentID != nil {
		pid := *c.ParentID
		cp.ParentID = &pid
	}
	return &cp
}
