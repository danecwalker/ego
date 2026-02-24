package ego

// Operation describes a database operation being executed.
type Operation struct {
	Type   string // "create", "update", "delete", "query"
	Entity any    // the entity being operated on (may be nil for queries)
	SQL    string // the generated SQL statement
	Args   []any  // the SQL arguments
}

// HandlerFunc executes a database operation.
type HandlerFunc func(op *Operation) error

// MiddlewareFunc wraps a HandlerFunc to add behavior before/after execution.
type MiddlewareFunc func(next HandlerFunc) HandlerFunc

// buildMiddlewareChain wraps the innermost handler with the registered
// middlewares. The chain is built in reverse order so that the first
// registered middleware is the outermost wrapper and executes first.
func buildMiddlewareChain(middlewares []MiddlewareFunc, inner HandlerFunc) HandlerFunc {
	handler := inner
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}
	return handler
}
