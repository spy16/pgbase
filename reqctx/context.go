package reqctx

import "context"

type reqCtxKeyType string

var reqCtxKey = reqCtxKeyType("req_ctx")

// Ctx returns a new Go context with reqCtx injected.
func Ctx(ctx context.Context, reqCtx ReqCtx) context.Context {
	return context.WithValue(ctx, reqCtxKey, reqCtx)
}

// From extracts the ReqCtx in the given Go context. Returns zero-value
// if not found.
func From(ctx context.Context) ReqCtx {
	v, _ := ctx.Value(reqCtxKey).(ReqCtx)
	return v
}
