package domain

type PostModel struct {
	PostId      string `json:"postId"`
	AuthorId    string `json:"authorId"`
	CreatedAt   string `json:"createdAt"`
	PublishedAt string `json:"publishedAt,omitempty"`
	UpdatedAt   string `json:"updatedAt"`
	Description string `json:"description"`
}

type PostsPage struct {
	Items      []PostModel `json:"items"`
	NextCursor string      `json:"nextCursor,omitempty"`
}
