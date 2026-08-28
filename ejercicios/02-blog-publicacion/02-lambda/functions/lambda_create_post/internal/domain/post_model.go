package domain

type PostStatus string

const (
	PostStatusDraft     PostStatus = "DRAFT"
	PostStatusPublished PostStatus = "PUBLISHED"
)

type PostModel struct {
	PostID      string     `json:"postId"`
	AuthorID    string     `json:"authorId"`
	Status      PostStatus `json:"status"`
	CreatedAt   string     `json:"createdAt"`
	PublishedAt string     `json:"publishedAt,omitempty"`
	UpdatedAt   string     `json:"updatedAt"`
	Description string     `json:"description"`
}

func (s PostStatus) IsValid() bool {
	return s == PostStatusDraft || s == PostStatusPublished
}
