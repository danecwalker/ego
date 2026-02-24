// executor_test.go
package ego_test

import (
	"testing"

	"github.com/danecwalker/ego"
	"github.com/danecwalker/ego/sqlite"
)

func TestDBImplementsExecutor(t *testing.T) {
	db, err := ego.Open(sqlite.New(":memory:"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// *DB must satisfy the Executor interface.
	var _ ego.Executor = db
}
