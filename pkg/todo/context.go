package todo

import "context"

type listKey struct{}

// WithList returns a context carrying the given list. Tool handlers resolve
// the session's list via ListFromContext.
func WithList(ctx context.Context, l *List) context.Context {
	return context.WithValue(ctx, listKey{}, l)
}

// ListFromContext extracts the list set by WithList, or returns nil.
func ListFromContext(ctx context.Context) *List {
	l, _ := ctx.Value(listKey{}).(*List)
	return l
}
