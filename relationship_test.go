// relationship_test.go
package ego_test

import (
	"context"
	"testing"

	"github.com/danewilson/ego"
)

type Author struct {
	ego.Model
	Name  string
	Posts []Post // HasMany
}

func (a *Author) Configure(b *ego.EntityBuilder[Author]) {
	b.ToTable("authors")
	b.Property(&a.Name).IsRequired()
	b.HasMany(&a.Posts)
}

type Post struct {
	ego.Model
	Title    string
	Body     string
	AuthorID int64
	Author   *Author // BelongsTo
}

func (p *Post) Configure(b *ego.EntityBuilder[Post]) {
	b.ToTable("posts")
	b.Property(&p.Title).IsRequired()
	b.BelongsTo(&p.Author)
}

func TestHasManyRelationshipRegistered(t *testing.T) {
	db := setupTestDB(t, &Author{}, &Post{})
	schema := ego.SchemaFor[Author](db)

	if len(schema.Relationships) == 0 {
		t.Fatal("expected relationships to be registered")
	}
	rel := schema.Relationships[0]
	if rel.Type != ego.HasManyRel {
		t.Errorf("expected HasMany, got %v", rel.Type)
	}
}

func TestIncludeHasManyLoadsChildren(t *testing.T) {
	db := setupTestDB(t, &Author{}, &Post{})
	ctx := context.Background()

	author := &Author{Name: "Alice"}
	ego.Create(db, ctx, author)
	ego.Create(db, ctx, &Post{Title: "Post 1", Body: "Body 1", AuthorID: author.ID})
	ego.Create(db, ctx, &Post{Title: "Post 2", Body: "Body 2", AuthorID: author.ID})

	result, err := ego.Query[Author](db, ctx).
		Where(ego.Col("id").Eq(author.ID)).
		Include("Posts").
		First()
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Posts) != 2 {
		t.Errorf("expected 2 posts, got %d", len(result.Posts))
	}
}

func TestBelongsToLoadsParent(t *testing.T) {
	db := setupTestDB(t, &Author{}, &Post{})
	ctx := context.Background()

	author := &Author{Name: "Alice"}
	ego.Create(db, ctx, author)
	ego.Create(db, ctx, &Post{Title: "Post 1", Body: "Body 1", AuthorID: author.ID})

	post, err := ego.Query[Post](db, ctx).
		Where(ego.Col("title").Eq("Post 1")).
		Include("Author").
		First()
	if err != nil {
		t.Fatal(err)
	}
	if post.Author == nil {
		t.Fatal("expected Author to be loaded")
	}
	if post.Author.Name != "Alice" {
		t.Errorf("expected Alice, got %s", post.Author.Name)
	}
}

func TestIncludeWithNoRelatedRecords(t *testing.T) {
	db := setupTestDB(t, &Author{}, &Post{})
	ctx := context.Background()

	author := &Author{Name: "NoPostsAuthor"}
	ego.Create(db, ctx, author)

	result, err := ego.Query[Author](db, ctx).
		Where(ego.Col("id").Eq(author.ID)).
		Include("Posts").
		First()
	if err != nil {
		t.Fatal(err)
	}
	if result.Posts == nil {
		// Should be empty slice, not nil
		t.Error("expected non-nil empty Posts slice")
	}
	if len(result.Posts) != 0 {
		t.Errorf("expected 0 posts, got %d", len(result.Posts))
	}
}
