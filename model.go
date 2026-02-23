package ego

import "time"

// Model provides base fields for entities. Embedding is optional —
// any struct with a configured primary key works.
type Model struct {
	ID        int64
	CreatedAt time.Time
	UpdatedAt time.Time
}
