package deployer

import (
	"context"
	"io/fs"
)

// Run is the root entry point for the Go deployer implementation.
func Run(ctx context.Context, bundle fs.FS, args []string) error {
	engine := NewEngine(bundle)
	app := newCLI(engine)
	argv := append([]string{"gochat-deployer"}, args...)
	return app.Run(ctx, argv)
}
