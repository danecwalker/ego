package ego

import "context"

// BeforeCreator is implemented by entities that need to run logic before
// being inserted into the database. The hook runs before timestamps are
// set and the INSERT is executed, so fields modified by the hook will be
// persisted.
type BeforeCreator interface {
	BeforeCreate(ctx context.Context) error
}

// AfterCreator is implemented by entities that need to run logic after
// being inserted into the database. The hook runs after the INSERT and
// the entity's ID has been populated from LastInsertId.
type AfterCreator interface {
	AfterCreate(ctx context.Context) error
}

// BeforeUpdater is implemented by entities that need to run logic before
// being updated in the database. The hook runs before UpdatedAt is set
// and the UPDATE is executed.
type BeforeUpdater interface {
	BeforeUpdate(ctx context.Context) error
}

// AfterUpdater is implemented by entities that need to run logic after
// being updated in the database.
type AfterUpdater interface {
	AfterUpdate(ctx context.Context) error
}

// BeforeDeleter is implemented by entities that need to run logic before
// being deleted from the database.
type BeforeDeleter interface {
	BeforeDelete(ctx context.Context) error
}

// AfterDeleter is implemented by entities that need to run logic after
// being deleted from the database.
type AfterDeleter interface {
	AfterDelete(ctx context.Context) error
}
