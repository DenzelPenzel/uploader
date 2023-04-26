package types

import "time"

type (
	ID          string
	Filename    string
	ContentType string
	Note        struct {
		Value *string
	}

	RecordMetadata struct {
		ID          ID
		Filename    Filename
		Note        Note
		ContentType ContentType
		CreateAt    time.Time
		Size        int64
	}

	RecordPostResponse struct {
		ID string `json:"id"`
	}
)
