// relationship_test.go
package ego_test

import (
	"context"
	"testing"

	"github.com/danewilson/ego"
)

type Profile struct {
	ego.Model
	Bio      string
	AuthorID int64
}

type Tag struct {
	ego.Model
	Label string
}

type Article struct {
	ego.Model
	Title string
	Tags  []Tag
}

func (a *Article) Configure(b *ego.EntityBuilder[Article]) {
	b.ToTable("articles")
	b.Property(&a.Title).IsRequired()
	b.ManyToMany(&a.Tags)
}

type Author struct {
	ego.Model
	Name    string
	Posts   []Post
	Profile *Profile
}

func (a *Author) Configure(b *ego.EntityBuilder[Author]) {
	b.ToTable("authors")
	b.Property(&a.Name).IsRequired()
	b.HasMany(&a.Posts)
	b.HasOne(&a.Profile)
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
	db := setupTestDB(t, &Author{}, &Post{}, &Profile{})
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

func TestHasOneLoadsRelated(t *testing.T) {
	db := setupTestDB(t, &Author{}, &Post{}, &Profile{})
	ctx := context.Background()

	author := &Author{Name: "Alice"}
	ego.Create(db, ctx, author)
	ego.Create(db, ctx, &Profile{Bio: "Writer", AuthorID: author.ID})

	result, err := ego.Query[Author](db, ctx).
		Where(ego.Col("id").Eq(author.ID)).
		Include("Profile").
		First()
	if err != nil {
		t.Fatal(err)
	}
	if result.Profile == nil {
		t.Fatal("expected Profile to be loaded")
	}
	if result.Profile.Bio != "Writer" {
		t.Errorf("expected 'Writer', got %q", result.Profile.Bio)
	}
}

func TestHasOneWithNoRelatedReturnsNil(t *testing.T) {
	db := setupTestDB(t, &Author{}, &Post{}, &Profile{})
	ctx := context.Background()

	author := &Author{Name: "NoProfile"}
	ego.Create(db, ctx, author)

	result, err := ego.Query[Author](db, ctx).
		Where(ego.Col("id").Eq(author.ID)).
		Include("Profile").
		First()
	if err != nil {
		t.Fatal(err)
	}
	if result.Profile != nil {
		t.Error("expected Profile to be nil")
	}
}

func TestManyToManyLoadsRelated(t *testing.T) {
	db := setupTestDB(t, &Article{}, &Tag{})
	ctx := context.Background()

	article := &Article{Title: "Go Generics"}
	ego.Create(db, ctx, article)

	tag1 := &Tag{Label: "golang"}
	tag2 := &Tag{Label: "generics"}
	ego.Create(db, ctx, tag1)
	ego.Create(db, ctx, tag2)

	// Associate via pivot table
	ego.Associate(db, ctx, article, tag1, tag2)

	result, err := ego.Query[Article](db, ctx).
		Where(ego.Col("id").Eq(article.ID)).
		Include("Tags").
		First()
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(result.Tags))
	}
}

func TestManyToManyWithNoAssociations(t *testing.T) {
	db := setupTestDB(t, &Article{}, &Tag{})
	ctx := context.Background()

	article := &Article{Title: "No Tags"}
	ego.Create(db, ctx, article)

	result, err := ego.Query[Article](db, ctx).
		Where(ego.Col("id").Eq(article.ID)).
		Include("Tags").
		First()
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Tags) != 0 {
		t.Errorf("expected 0 tags, got %d", len(result.Tags))
	}
}
