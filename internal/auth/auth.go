package auth

import (
	"crypto/sha1"
	"encoding/hex"
	"path/filepath"
	"time"

	"moyuan-code/internal/fsutil"
	"moyuan-code/internal/logging"
	"moyuan-code/internal/workspace"
)

type Owner struct {
	ActorID     string `json:"actor_id"`
	DisplayName string `json:"display_name"`
	Mode        string `json:"mode"`
	CreatedAt   string `json:"created_at"`
}

type Context struct {
	ActorID   string `json:"actor_id"`
	Action    string `json:"action"`
	Decision  string `json:"decision"`
	Risk      string `json:"risk"`
	CreatedAt string `json:"created_at"`
}

func ownerPath(rootDir string) string {
	return filepath.Join(workspace.ForRoot(rootDir).AuthDir, "owner.json")
}

func makeOwner(rootDir string, name string) Owner {
	sum := sha1.Sum([]byte(rootDir + ":" + name))
	return Owner{
		ActorID:     "owner-" + hex.EncodeToString(sum[:])[:12],
		DisplayName: name,
		Mode:        "local_single_user",
		CreatedAt:   time.Now().UTC().Format(time.RFC3339Nano),
	}
}

func InitOwner(rootDir string, name string) (Owner, error) {
	if name == "" {
		name = "local-owner"
	}
	owner := makeOwner(rootDir, name)
	if err := fsutil.WriteJSON(ownerPath(rootDir), owner); err != nil {
		return Owner{}, err
	}
	ws, err := workspace.Load(rootDir)
	if err != nil {
		return Owner{}, err
	}
	ws.Access.Access.Mode = "local_single_user"
	ws.Access.Access.LocalOwnerID = &owner.ActorID
	ws.Access.Access.OrganizationID = nil
	if ws.Access.Access.ProjectRoles == nil {
		ws.Access.Access.ProjectRoles = map[string][]string{"owner": {"*"}}
	}
	if err := workspace.SaveAccess(rootDir, ws.Access); err != nil {
		return Owner{}, err
	}
	_ = logging.Log(rootDir, "audit", "auth.owner.initialized", map[string]any{
		"actor_id":     owner.ActorID,
		"display_name": owner.DisplayName,
	})
	return owner, nil
}

func Whoami(rootDir string) (Owner, error) {
	var owner Owner
	found, err := fsutil.ReadJSON(ownerPath(rootDir), &owner)
	if err != nil {
		return Owner{}, err
	}
	if !found {
		return Owner{ActorID: "anonymous", DisplayName: "anonymous", Mode: "unknown"}, nil
	}
	return owner, nil
}

func NewContext(rootDir string, action string, risk string) (Context, error) {
	if risk == "" {
		risk = "normal"
	}
	owner, err := Whoami(rootDir)
	if err != nil {
		return Context{}, err
	}
	decision := "ALLOW"
	highRisk := map[string]bool{
		"git.push":        true,
		"git.tag":         true,
		"release.publish": true,
		"deploy.run":      true,
		"server.write":    true,
	}
	if highRisk[action] {
		decision = "REQUIRE_APPROVAL"
	}
	ctx := Context{
		ActorID:   owner.ActorID,
		Action:    action,
		Decision:  decision,
		Risk:      risk,
		CreatedAt: time.Now().UTC().Format(time.RFC3339Nano),
	}
	_ = logging.Log(rootDir, "audit", "auth.context.created", map[string]any{
		"actor_id": ctx.ActorID,
		"action":   ctx.Action,
		"decision": ctx.Decision,
		"risk":     ctx.Risk,
	})
	return ctx, nil
}
