package controller

import "github.com/SodaTeaaaaee/EliGiftManager/internal/app"

type WorkspaceController struct {
	ws *app.Workspace
}

func NewWorkspaceController(ws *app.Workspace) *WorkspaceController {
	return &WorkspaceController{ws: ws}
}
