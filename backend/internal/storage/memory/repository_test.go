package memory

import (
	"context"
	"errors"
	"testing"

	"posts_service/internal/repository"
)

func TestCreatePostAndCommentsHierarchy(t *testing.T) {
	ctx := context.Background()
	repo := NewRepository()

	u, err := repo.CreateUser(ctx, "a@test.com", "alice", "hash")
	if err != nil {
		t.Fatal(err)
	}

	post, err := repo.CreatePost(ctx, u.ID, "hello post", true)
	if err != nil {
		t.Fatal(err)
	}

	root, err := repo.CreateComment(ctx, u.ID, post.ID, nil, "root comment")
	if err != nil {
		t.Fatal(err)
	}

	reply, err := repo.CreateComment(ctx, u.ID, post.ID, &root.ID, "nested reply")
	if err != nil {
		t.Fatal(err)
	}

	deep, err := repo.CreateComment(ctx, u.ID, post.ID, &reply.ID, "deep nested")
	if err != nil {
		t.Fatal(err)
	}

	roots, err := repo.ListRootComments(ctx, post.ID, 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(roots) != 1 || roots[0].ID != root.ID {
		t.Fatalf("expected 1 root comment, got %+v", roots)
	}

	replies, err := repo.ListReplyComments(ctx, root.ID, 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(replies) != 1 || replies[0].ID != reply.ID {
		t.Fatalf("expected 1 reply, got %+v", replies)
	}

	deepReplies, err := repo.ListReplyComments(ctx, reply.ID, 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(deepReplies) != 1 || deepReplies[0].ID != deep.ID {
		t.Fatalf("expected 1 deep reply, got %+v", deepReplies)
	}
}

func TestCommentsDisabled(t *testing.T) {
	ctx := context.Background()
	repo := NewRepository()

	u, _ := repo.CreateUser(ctx, "b@test.com", "bob", "hash")
	post, _ := repo.CreatePost(ctx, u.ID, "no comments", false)

	_, err := repo.CreateComment(ctx, u.ID, post.ID, nil, "should fail")
	if !errors.Is(err, repository.ErrCommentsDisabled) {
		t.Fatalf("expected ErrCommentsDisabled, got %v", err)
	}
}

func TestCommentMaxLength(t *testing.T) {
	ctx := context.Background()
	repo := NewRepository()

	u, _ := repo.CreateUser(ctx, "c@test.com", "carol", "hash")
	post, _ := repo.CreatePost(ctx, u.ID, "post", true)

	longText := make([]rune, repository.MaxCommentLength+1)
	for i := range longText {
		longText[i] = 'x'
	}

	_, err := repo.CreateComment(ctx, u.ID, post.ID, nil, string(longText))
	if !errors.Is(err, repository.ErrCommentTooLong) {
		t.Fatalf("expected ErrCommentTooLong, got %v", err)
	}
}

func TestUpdatePostCommentsAllowedOnlyOwner(t *testing.T) {
	ctx := context.Background()
	repo := NewRepository()

	owner, _ := repo.CreateUser(ctx, "owner@test.com", "owner", "hash")
	other, _ := repo.CreateUser(ctx, "other@test.com", "other", "hash")
	post, _ := repo.CreatePost(ctx, owner.ID, "owned", true)

	_, err := repo.UpdatePostCommentsAllowed(ctx, other.ID, post.ID, false)
	if !errors.Is(err, repository.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}

	updated, err := repo.UpdatePostCommentsAllowed(ctx, owner.ID, post.ID, false)
	if err != nil {
		t.Fatal(err)
	}
	if updated.CanHaveComm {
		t.Fatal("expected comments disabled")
	}
}

func TestCommentPagination(t *testing.T) {
	ctx := context.Background()
	repo := NewRepository()

	u, _ := repo.CreateUser(ctx, "d@test.com", "dan", "hash")
	post, _ := repo.CreatePost(ctx, u.ID, "paginated", true)

	for i := 0; i < 5; i++ {
		if _, err := repo.CreateComment(ctx, u.ID, post.ID, nil, "c"); err != nil {
			t.Fatal(err)
		}
	}

	page1, err := repo.ListRootComments(ctx, post.ID, 2, 0)
	if err != nil || len(page1) != 2 {
		t.Fatalf("page1: len=%d err=%v", len(page1), err)
	}
	page2, err := repo.ListRootComments(ctx, post.ID, 2, 2)
	if err != nil || len(page2) != 2 {
		t.Fatalf("page2: len=%d err=%v", len(page2), err)
	}
	page3, err := repo.ListRootComments(ctx, post.ID, 2, 4)
	if err != nil || len(page3) != 1 {
		t.Fatalf("page3: len=%d err=%v", len(page3), err)
	}
}

func TestInvalidParentComment(t *testing.T) {
	ctx := context.Background()
	repo := NewRepository()

	u, _ := repo.CreateUser(ctx, "e@test.com", "eve", "hash")
	post1, _ := repo.CreatePost(ctx, u.ID, "post1", true)
	post2, _ := repo.CreatePost(ctx, u.ID, "post2", true)
	c1, _ := repo.CreateComment(ctx, u.ID, post1.ID, nil, "on post1")

	wrongParent := c1.ID
	_, err := repo.CreateComment(ctx, u.ID, post2.ID, &wrongParent, "wrong post")
	if !errors.Is(err, repository.ErrInvalidParent) {
		t.Fatalf("expected ErrInvalidParent, got %v", err)
	}
}
