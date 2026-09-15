package domain

type FanoutJob struct { EventID string `json:"eventId"`; FanoutID string `json:"fanoutId"`; JobID string `json:"jobId"`; StreamEventID string `json:"streamEventId"`; PostID string `json:"postId"`; AuthorID string `json:"authorId"`; Cursor string `json:"cursor"`; BatchNumber int `json:"batchNumber"`; CreatedAt string `json:"createdAt"` }
type NotificationJob struct { EventID string `json:"eventId"`; FanoutBatchID string `json:"fanoutBatchId"`; PostID string `json:"postId"`; AuthorID string `json:"authorId"`; RecipientIDs []string `json:"recipientIds"`; CreatedAt string `json:"createdAt"` }
type FollowerPage struct { RecipientIDs []string; NextCursor string }
type ProgressBatch struct { BatchID string; CursorStart string; CursorEnd string; RecipientsCount int; CreatedAt string }
type FanoutCheckpoint struct { NextCursor string; Completed bool }
