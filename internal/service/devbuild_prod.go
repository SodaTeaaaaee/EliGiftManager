//go:build production

package service

// isDevBuild reports whether the binary was built without the `production`
// build tag. `wails3 dev` and plain `go build`/`go test` produce dev builds;
// `wails3 build` compiles with -tags production.
func isDevBuild() bool { return false }
