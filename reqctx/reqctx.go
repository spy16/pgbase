package reqctx

import (
	"github.com/spy16/pgbase/auth"
)

// ReqCtx represents the contextual information in the current request
// context.
type ReqCtx struct {
	ReqID      string
	Token      string
	CurUser    *auth.User
	ClientAddr string
}
