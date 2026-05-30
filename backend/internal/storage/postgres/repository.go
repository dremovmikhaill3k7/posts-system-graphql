package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"posts_service/internal/graph/model"
	"posts_service/internal/repository"
	"posts_service/internal/utils"

	"github.com/lib/pq"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreateUser(ctx context.Context, email, username, passwordHash string) (*model.User, error) {
	var u model.User
	var createdAt time.Time
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO users (email, name, password, status)
		VALUES ($1, $2, $3, 'active')
		RETURNING id, name, email, status, created_at`,
		email, username, passwordHash,
	).Scan(&u.ID, &u.Username, &u.Email, &u.Status, &createdAt)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return nil, repository.ErrEmailTaken
		}
		return nil, err
	}
	u.CreatedAt = &createdAt
	return &u, nil
}

func (r *Repository) GetUserByEmail(ctx context.Context, email string) (*model.User, string, error) {
	var u model.User
	var passwordHash string
	var createdAt time.Time
	err := r.db.QueryRowContext(ctx, `
		SELECT id, name, email, password, status, created_at
		FROM users WHERE email = $1`, email,
	).Scan(&u.ID, &u.Username, &u.Email, &passwordHash, &u.Status, &createdAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, "", repository.ErrNotFound
	}
	if err != nil {
		return nil, "", err
	}
	u.CreatedAt = &createdAt
	return &u, passwordHash, nil
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

func (r *Repository) GetUsersByIDs(ctx context.Context, ids []int) (map[int]*model.User, error) {
	if len(ids) == 0 {
		return make(map[int]*model.User), nil
	}
	placeholders, args := utils.CreatePlaceholders(ids)
	query := fmt.Sprintf(`
		SELECT id, name, email, status, created_at
		FROM users WHERE id IN (%s)`, strings.Join(placeholders, ","))

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[int]*model.User, len(ids))
	for rows.Next() {
		var u model.User
		var createdAt time.Time
		if err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.Status, &createdAt); err != nil {
			return nil, err
		}
		u.CreatedAt = &createdAt
		result[u.ID] = &u
	}
	return result, rows.Err()
}

func (r *Repository) ListUsers(ctx context.Context, limit, offset int, search string) ([]*model.User, error) {
	query := `SELECT id, name, email, status, created_at FROM users WHERE status = 'active'`
	args := []any{}
	argN := 1
	if search != "" {
		query += fmt.Sprintf(` AND (name ILIKE $%d OR email ILIKE $%d)`, argN, argN)
		args = append(args, "%"+search+"%")
		argN++
	}
	query += fmt.Sprintf(` ORDER BY id LIMIT $%d OFFSET $%d`, argN, argN+1)
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanUsers(rows)
}

func (r *Repository) CreatePost(ctx context.Context, userID int, text string, canHaveComm bool) (*model.Post, error) {
	var p model.Post
	var createdAt, updatedAt time.Time
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO posts (user_id, text, can_have_comm, status)
		VALUES ($1, $2, $3, 'active')
		RETURNING id, user_id, text, can_have_comm, status, created_at, updated_at`,
		userID, text, canHaveComm,
	).Scan(&p.ID, &userID, &p.Text, &p.CanHaveComm, &p.Status, &createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}
	p.User = &model.User{ID: userID}
	p.CreatedAt = &createdAt
	p.UpdatedAt = &updatedAt
	return &p, nil
}

func (r *Repository) GetPostsByIDs(ctx context.Context, ids []int) (map[int]*model.Post, error) {
	if len(ids) == 0 {
		return make(map[int]*model.Post), nil
	}
	placeholders, args := utils.CreatePlaceholders(ids)
	query := fmt.Sprintf(`
		SELECT id, user_id, text, can_have_comm, status, created_at, updated_at
		FROM posts WHERE id IN (%s) AND status = 'active'`, strings.Join(placeholders, ","))

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[int]*model.Post, len(ids))
	posts, err := scanPosts(rows)
	if err != nil {
		return nil, err
	}
	for _, p := range posts {
		result[p.ID] = p
	}
	return result, nil
}

func (r *Repository) GetPostByID(ctx context.Context, id int) (*model.Post, error) {
	var p model.Post
	var userID int
	var createdAt, updatedAt time.Time
	err := r.db.QueryRowContext(ctx, `
		SELECT id, user_id, text, can_have_comm, status, created_at, updated_at
		FROM posts WHERE id = $1 AND status = 'active'`, id,
	).Scan(&p.ID, &userID, &p.Text, &p.CanHaveComm, &p.Status, &createdAt, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, repository.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	p.User = &model.User{ID: userID}
	p.CreatedAt = &createdAt
	p.UpdatedAt = &updatedAt
	return &p, nil
}

func (r *Repository) ListPosts(ctx context.Context, limit, offset int, search string) ([]*model.Post, error) {
	query := `SELECT id, user_id, text, can_have_comm, status, created_at, updated_at
		FROM posts WHERE status = 'active'`
	args := []any{}
	argN := 1
	if search != "" {
		query += fmt.Sprintf(` AND text ILIKE $%d`, argN)
		args = append(args, "%"+search+"%")
		argN++
	}
	query += fmt.Sprintf(` ORDER BY created_at DESC LIMIT $%d OFFSET $%d`, argN, argN+1)
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPosts(rows)
}

func (r *Repository) ListPostsByUser(ctx context.Context, userID, limit, offset int) ([]*model.Post, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, user_id, text, can_have_comm, status, created_at, updated_at
		FROM posts WHERE user_id = $1 AND status = 'active'
		ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		userID, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPosts(rows)
}

func (r *Repository) UpdatePostCommentsAllowed(ctx context.Context, userID, postID int, canHaveComm bool) (*model.Post, error) {
	var ownerID int
	err := r.db.QueryRowContext(ctx, `SELECT user_id FROM posts WHERE id = $1 AND status = 'active'`, postID).Scan(&ownerID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, repository.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if ownerID != userID {
		return nil, repository.ErrForbidden
	}

	var p model.Post
	var createdAt, updatedAt time.Time
	err = r.db.QueryRowContext(ctx, `
		UPDATE posts SET can_have_comm = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2
		RETURNING id, user_id, text, can_have_comm, status, created_at, updated_at`,
		canHaveComm, postID,
	).Scan(&p.ID, &ownerID, &p.Text, &p.CanHaveComm, &p.Status, &createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}
	p.User = &model.User{ID: ownerID}
	p.CreatedAt = &createdAt
	p.UpdatedAt = &updatedAt
	return &p, nil
}

func (r *Repository) CreateComment(ctx context.Context, userID, postID int, parentID *int, text string) (*model.Comment, error) {
	if len([]rune(text)) > repository.MaxCommentLength {
		return nil, repository.ErrCommentTooLong
	}

	post, err := r.GetPostByID(ctx, postID)
	if err != nil {
		return nil, err
	}
	if !post.CanHaveComm {
		return nil, repository.ErrCommentsDisabled
	}

	if parentID != nil {
		var parentPostID int
		err := r.db.QueryRowContext(ctx, `
			SELECT post_id FROM comments WHERE id = $1 AND status = 'active'`, *parentID,
		).Scan(&parentPostID)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repository.ErrInvalidParent
		}
		if err != nil {
			return nil, err
		}
		if parentPostID != postID {
			return nil, repository.ErrInvalidParent
		}
	}

	var c model.Comment
	var createdAt, updatedAt time.Time
	err = r.db.QueryRowContext(ctx, `
		INSERT INTO comments (user_id, post_id, parent_id, text, status)
		VALUES ($1, $2, $3, $4, 'active')
		RETURNING id, parent_id, text, status, created_at, updated_at`,
		userID, postID, parentID, text,
	).Scan(&c.ID, &c.ParentID, &c.Text, &c.Status, &createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}
	c.User = &model.User{ID: userID}
	c.Post = &model.Post{ID: postID}
	c.CreatedAt = &createdAt
	c.UpdatedAt = &updatedAt
	return &c, nil
}

func (r *Repository) ListRootCommentsForPosts(ctx context.Context, postIDs []int, limit, offset int) (map[int][]*model.Comment, error) {
	result := make(map[int][]*model.Comment, len(postIDs))
	if len(postIDs) == 0 {
		return result, nil
	}
	for _, id := range postIDs {
		result[id] = []*model.Comment{}
	}

	placeholders, args := utils.CreatePlaceholders(postIDs)
	query := fmt.Sprintf(`
		SELECT id, user_id, post_id, parent_id, text, status, created_at, updated_at
		FROM comments
		WHERE post_id IN (%s) AND parent_id IS NULL AND status = 'active'
		ORDER BY post_id, created_at ASC`, strings.Join(placeholders, ","))

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	grouped := make(map[int][]*model.Comment, len(postIDs))
	for rows.Next() {
		var c model.Comment
		var userID, postID int
		var createdAt, updatedAt time.Time
		if err := rows.Scan(&c.ID, &userID, &postID, &c.ParentID, &c.Text, &c.Status, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		c.User = &model.User{ID: userID}
		c.Post = &model.Post{ID: postID}
		c.CreatedAt = &createdAt
		c.UpdatedAt = &updatedAt
		grouped[postID] = append(grouped[postID], &c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for postID, list := range grouped {
		if offset >= len(list) {
			result[postID] = []*model.Comment{}
			continue
		}
		end := offset + limit
		if end > len(list) {
			end = len(list)
		}
		result[postID] = list[offset:end]
	}
	return result, nil
}

func (r *Repository) ListRootComments(ctx context.Context, postID, limit, offset int) ([]*model.Comment, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, user_id, post_id, parent_id, text, status, created_at, updated_at
		FROM comments
		WHERE post_id = $1 AND parent_id IS NULL AND status = 'active'
		ORDER BY created_at ASC
		LIMIT $2 OFFSET $3`, postID, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanComments(rows)
}

func (r *Repository) ListReplyComments(ctx context.Context, parentID, limit, offset int) ([]*model.Comment, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, user_id, post_id, parent_id, text, status, created_at, updated_at
		FROM comments
		WHERE parent_id = $1 AND status = 'active'
		ORDER BY created_at ASC
		LIMIT $2 OFFSET $3`, parentID, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanComments(rows)
}

func (r *Repository) CommentsHaveReplies(ctx context.Context, commentIDs []int) (map[int]bool, error) {
	result := make(map[int]bool, len(commentIDs))
	for _, id := range commentIDs {
		result[id] = false
	}
	if len(commentIDs) == 0 {
		return result, nil
	}

	placeholders, args := utils.CreatePlaceholders(commentIDs)
	query := fmt.Sprintf(`
		SELECT DISTINCT parent_id FROM comments
		WHERE parent_id IN (%s) AND status = 'active'`, strings.Join(placeholders, ","))

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var parentID int
		if err := rows.Scan(&parentID); err != nil {
			return nil, err
		}
		result[parentID] = true
	}
	return result, rows.Err()
}

func scanUsers(rows *sql.Rows) ([]*model.User, error) {
	var users []*model.User
	for rows.Next() {
		var u model.User
		var createdAt time.Time
		if err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.Status, &createdAt); err != nil {
			return nil, err
		}
		u.CreatedAt = &createdAt
		users = append(users, &u)
	}
	return users, rows.Err()
}

func scanPosts(rows *sql.Rows) ([]*model.Post, error) {
	var posts []*model.Post
	for rows.Next() {
		var p model.Post
		var userID int
		var createdAt, updatedAt time.Time
		if err := rows.Scan(&p.ID, &userID, &p.Text, &p.CanHaveComm, &p.Status, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		p.User = &model.User{ID: userID}
		p.CreatedAt = &createdAt
		p.UpdatedAt = &updatedAt
		posts = append(posts, &p)
	}
	return posts, rows.Err()
}

func scanComments(rows *sql.Rows) ([]*model.Comment, error) {
	var comments []*model.Comment
	for rows.Next() {
		var c model.Comment
		var userID, postID int
		var createdAt, updatedAt time.Time
		if err := rows.Scan(&c.ID, &userID, &postID, &c.ParentID, &c.Text, &c.Status, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		c.User = &model.User{ID: userID}
		c.Post = &model.Post{ID: postID}
		c.CreatedAt = &createdAt
		c.UpdatedAt = &updatedAt
		comments = append(comments, &c)
	}
	return comments, rows.Err()
}
