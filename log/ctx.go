package log

import (
	"context"
)

type ctxKey string

var fieldsKey = ctxKey("fields")

type Fields map[string]any

// Ctx returns a new context with fields injected.
func Ctx(ctx context.Context, fields Fields) context.Context {
	return context.WithValue(ctx, fieldsKey, fields)
}

func fromCtx(ctx context.Context) Fields {
	f, _ := ctx.Value(fieldsKey).(Fields)
	return f
}
