// dialect_test.go
package ego_test

import (
	"testing"

	"github.com/danewilson/ego"
)

// mockDialect verifies the interface is implementable
type mockDialect struct{}

func (d *mockDialect) Name() string                       { return "mock" }
func (d *mockDialect) Placeholder(index int) string       { return "?" }
func (d *mockDialect) QuoteIdentifier(name string) string { return `"` + name + `"` }
func (d *mockDialect) AutoIncrementDef() string           { return "AUTOINCREMENT" }
func (d *mockDialect) TypeMapping(goType string) string   { return "TEXT" }
func (d *mockDialect) SupportsReturning() bool            { return false }

func TestDialectInterface(t *testing.T) {
	var d ego.Dialect = &mockDialect{}
	if d.Name() != "mock" {
		t.Errorf("expected mock, got %s", d.Name())
	}
	if d.Placeholder(1) != "?" {
		t.Errorf("expected ?, got %s", d.Placeholder(1))
	}
	if d.QuoteIdentifier("users") != `"users"` {
		t.Errorf("unexpected quote result")
	}
}
