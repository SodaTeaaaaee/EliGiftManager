package controller

import "context"

// appContext holds the Wails application context captured at startup. It is the
// root context for all controller-initiated backend operations, so in-flight
// work is cancelled when the application shuts down.
//
// Before startup runs — e.g. in tests that construct controllers directly — it
// defaults to context.Background(), which is never cancelled.
var appContext = context.Background()

// SetAppContext stores the Wails application context. Called once from
// App.startup.
func SetAppContext(ctx context.Context) {
	appContext = ctx
}
