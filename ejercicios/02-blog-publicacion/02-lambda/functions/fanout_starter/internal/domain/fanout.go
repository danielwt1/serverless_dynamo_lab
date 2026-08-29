package domain

type FanoutJob struct {
	EventID       string `json:"eventId"`
	StreamEventID string `json:"streamEventId"`
	PostID        string `json:"postId"`
	AuthorID      string `json:"authorId"`
	Cursor        string `json:"cursor"`
	BatchNumber   int    `json:"batchNumber"`
	CreatedAt     string `json:"createdAt"`
}

type PublishTransition struct {
	EventID  string
	PostID   string
	AuthorID string
}
