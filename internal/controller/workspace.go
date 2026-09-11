package controller

import (
	"context"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app"
	application "github.com/wailsapp/wails/v3/pkg/application"
)

type WorkspaceController struct {
	ws  *app.Workspace
	ctx context.Context
}

func NewWorkspaceController(ws *app.Workspace) *WorkspaceController {
	return &WorkspaceController{ws: ws, ctx: context.Background()}
}

// ServiceStartup receives the Wails application context, valid until the app
// shuts down. It is the root context for controller-initiated backend work so
// in-flight operations are cancelled on shutdown. Controllers constructed
// directly (e.g. in tests) keep the context.Background() default.
func (c *WorkspaceController) ServiceStartup(ctx context.Context, options application.ServiceOptions) error {
	c.ctx = ctx
	return nil
}
