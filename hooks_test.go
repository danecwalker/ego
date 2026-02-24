package ego_test

import (
	"context"
	"strings"
	"testing"

	"github.com/danecwalker/ego"
)

type HookedEntity struct {
	ego.Model
	Email   string
	HookLog string
}

func (h *HookedEntity) BeforeCreate(ctx context.Context) error {
	h.Email = strings.ToLower(h.Email)
	h.HookLog += "before_create;"
	return nil
}

func (h *HookedEntity) AfterCreate(ctx context.Context) error {
	h.HookLog += "after_create;"
	return nil
}

func (h *HookedEntity) BeforeUpdate(ctx context.Context) error {
	h.HookLog += "before_update;"
	return nil
}

func (h *HookedEntity) AfterUpdate(ctx context.Context) error {
	h.HookLog += "after_update;"
	return nil
}

func (h *HookedEntity) BeforeDelete(ctx context.Context) error {
	h.HookLog += "before_delete;"
	return nil
}

func (h *HookedEntity) AfterDelete(ctx context.Context) error {
	h.HookLog += "after_delete;"
	return nil
}

func TestBeforeCreateHookModifiesEntity(t *testing.T) {
	db := setupTestDB(t, &HookedEntity{})
	ctx := context.Background()

	e := &HookedEntity{Email: "ALICE@EXAMPLE.COM"}
	ego.Create(db, ctx, e)

	if e.Email != "alice@example.com" {
		t.Errorf("expected lowercase email, got %q", e.Email)
	}
}

func TestCreateHookOrder(t *testing.T) {
	db := setupTestDB(t, &HookedEntity{})
	ctx := context.Background()

	e := &HookedEntity{Email: "test@test.com"}
	ego.Create(db, ctx, e)

	if e.HookLog != "before_create;after_create;" {
		t.Errorf("unexpected hook log: %q", e.HookLog)
	}
}

func TestUpdateHooksFire(t *testing.T) {
	db := setupTestDB(t, &HookedEntity{})
	ctx := context.Background()

	e := &HookedEntity{Email: "test@test.com"}
	ego.Create(db, ctx, e)
	e.HookLog = ""

	e.Email = "updated@test.com"
	ego.Update(db, ctx, e)

	if e.HookLog != "before_update;after_update;" {
		t.Errorf("unexpected hook log: %q", e.HookLog)
	}
}

func TestDeleteHooksFire(t *testing.T) {
	db := setupTestDB(t, &HookedEntity{})
	ctx := context.Background()

	e := &HookedEntity{Email: "test@test.com"}
	ego.Create(db, ctx, e)
	e.HookLog = ""

	ego.Delete(db, ctx, e)

	if e.HookLog != "before_delete;after_delete;" {
		t.Errorf("unexpected hook log: %q", e.HookLog)
	}
}
