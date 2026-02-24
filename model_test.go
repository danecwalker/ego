// model_test.go
package ego_test

import (
	"testing"
	"time"

	"github.com/danecwalker/ego"
)

func TestModelHasRequiredFields(t *testing.T) {
	m := ego.Model{}
	if m.ID != 0 {
		t.Errorf("expected zero ID, got %d", m.ID)
	}
	if !m.CreatedAt.IsZero() {
		t.Errorf("expected zero CreatedAt")
	}
	if !m.UpdatedAt.IsZero() {
		t.Errorf("expected zero UpdatedAt")
	}
}

func TestModelFieldsSettable(t *testing.T) {
	now := time.Now()
	m := ego.Model{ID: 42, CreatedAt: now, UpdatedAt: now}
	if m.ID != 42 {
		t.Errorf("expected ID 42, got %d", m.ID)
	}
	if !m.CreatedAt.Equal(now) {
		t.Errorf("expected CreatedAt %v, got %v", now, m.CreatedAt)
	}
}
