package httpx

import (
	"context"
	"net/http"
	"time"

	"github.com/spy16/pgbase/errors"
	"github.com/spy16/pgbase/log"
)

// HandlerFuncE is an extended version of http.HandlerFunc with automatic error
// handling.
func HandlerFuncE(fn func(w http.ResponseWriter, r *http.Request) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := fn(w, r)
		if err != nil {
			WriteErr(w, r, err)
		}
	}
}

// GracefulServe serves the given handler on 'addr'. Blocks until server
// exits due to critical error. On context-cancellation, server will be
// shutdown gracefully.
func GracefulServe(ctx context.Context, addr string, r http.Handler, graceT time.Duration) error {
	errCh := make(chan error)

	if graceT == 0 {
		graceT = 5 * time.Second
	}

	httpServer := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	go func() {
		err := httpServer.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err

	case <-ctx.Done():
		graceCtx, cancel := context.WithTimeout(context.Background(), graceT)
		defer cancel()

		log.Info(graceCtx, "shutting down server", log.Fields{"reason": ctx.Err()})
		if err := httpServer.Shutdown(graceCtx); err != nil {
			return err
		}
		log.Info(graceCtx, "server shutdown complete")
		return nil
	}
}
