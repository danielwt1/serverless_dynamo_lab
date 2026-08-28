package application

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strings"
	"time"

	"lambda_create_post/internal/domain"
	"lambda_create_post/internal/domain/ports/out"
)

type CreatePostUseCase struct {
	repository out.CreatePostPort
	now        func() time.Time
	newID      func() (string, error)
}

func NewCreatePostUseCase(repository out.CreatePostPort) *CreatePostUseCase {
	return &CreatePostUseCase{repository: repository, now: time.Now, newID: newPostID}
}

func (uc *CreatePostUseCase) CreatePost(ctx context.Context, authorID, description string, status domain.PostStatus) (domain.PostModel, error) {
	authorID = strings.TrimSpace(authorID)
	description = strings.TrimSpace(description)
	if authorID == "" {
		return domain.PostModel{}, domain.ErrInvalidAuthorID
	}
	if description == "" {
		return domain.PostModel{}, domain.ErrInvalidDescription
	}
	if !status.IsValid() {
		return domain.PostModel{}, domain.ErrInvalidStatus
	}

	postID, err := uc.newID()
	if err != nil {
		return domain.PostModel{}, err
	}
	now := uc.now().UTC().Format(time.RFC3339Nano)
	post := domain.PostModel{PostID: postID, AuthorID: authorID, Status: status, CreatedAt: now, UpdatedAt: now, Description: description}
	if status == domain.PostStatusPublished {
		post.PublishedAt = now
	}
	if err := uc.repository.SavePost(ctx, post); err != nil {
		return domain.PostModel{}, err
	}
	return post, nil
}

func newPostID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	// UUID v4, without adding an external dependency.
	bytes[6] = (bytes[6] & 0x0f) | 0x40
	bytes[8] = (bytes[8] & 0x3f) | 0x80
	encoded := hex.EncodeToString(bytes)
	return encoded[0:8] + "-" + encoded[8:12] + "-" + encoded[12:16] + "-" + encoded[16:20] + "-" + encoded[20:32], nil
}
