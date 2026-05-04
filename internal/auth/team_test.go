package auth

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"moyuan-code/internal/workspace"
)

func TestTeamAuthSessionTokenAndServiceAccountLifecycle(t *testing.T) {
	root := t.TempDir()
	if _, err := workspace.Ensure(root); err != nil {
		t.Fatal(err)
	}

	session, err := CreateSession(root, CreateSessionOptions{UserID: "Alice", DisplayName: "Alice", Roles: []string{"developer", "reviewer"}})
	if err != nil {
		t.Fatal(err)
	}
	if session.Status != "active" || session.UserID != "alice" {
		t.Fatalf("unexpected session: %+v", session)
	}
	sessions, err := ListSessions(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 1 || sessions[0].ID != session.ID {
		t.Fatalf("expected session in list: %+v", sessions)
	}
	revokedSession, found, err := RevokeSession(root, session.ID, RevokeOptions{ActorID: "owner", Reason: "test cleanup"})
	if err != nil {
		t.Fatal(err)
	}
	if !found || revokedSession.Status != "revoked" {
		t.Fatalf("expected revoked session, found=%v session=%+v", found, revokedSession)
	}

	created, err := CreateAPIToken(root, CreateTokenOptions{Name: "automation", ActorID: "svc-ci", Scopes: []string{"project:read", "deploy:dry-run"}})
	if err != nil {
		t.Fatal(err)
	}
	if created.TokenValue == "" || !strings.HasPrefix(created.TokenValue, "moyuan_") {
		t.Fatalf("expected one-time token value, got %+v", created)
	}
	requestContext, ok, err := ResolveBearer(root, "Bearer "+created.TokenValue)
	if err != nil {
		t.Fatal(err)
	}
	if !ok || requestContext.APITokenID != created.Token.ID {
		t.Fatalf("expected bearer token to resolve, ok=%v ctx=%+v", ok, requestContext)
	}
	allowed := Authorize(requestContext, "deploy.dry_run", "medium", []string{"deploy:dry-run"})
	if allowed.Decision != "ALLOW" {
		t.Fatalf("expected token scope to allow deploy dry-run: %+v", allowed)
	}
	denied := Authorize(requestContext, "resource.renew", "high", []string{"resource:write"})
	if denied.Decision != "DENY" || denied.Reason != "AUTH_TOKEN_SCOPE_MISMATCH" {
		t.Fatalf("expected missing scope denial: %+v", denied)
	}
	tokens, err := ListAPITokens(root)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(tokens)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), created.TokenValue) || strings.Contains(string(encoded), "token_hash") {
		t.Fatalf("listed tokens leaked token value or hash: %s", string(encoded))
	}
	revokedToken, found, err := RevokeAPIToken(root, created.Token.ID, RevokeOptions{ActorID: "owner"})
	if err != nil {
		t.Fatal(err)
	}
	if !found || revokedToken.Status != "revoked" {
		t.Fatalf("expected revoked token, found=%v token=%+v", found, revokedToken)
	}

	serviceAccount, err := CreateServiceAccount(root, CreateServiceAccountOptions{Name: "Release Bot", Roles: []string{"release_bot"}})
	if err != nil {
		t.Fatal(err)
	}
	if serviceAccount.ID != "svc-release-bot" || serviceAccount.Status != "active" {
		t.Fatalf("unexpected service account: %+v", serviceAccount)
	}
	accounts, err := ListServiceAccounts(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(accounts) != 1 || accounts[0].ID != serviceAccount.ID {
		t.Fatalf("expected service account in list: %+v", accounts)
	}

	data, err := os.ReadFile(teamStatePath(root))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), created.TokenValue) {
		t.Fatalf("team state leaked raw token: %s", string(data))
	}
	if !strings.Contains(string(data), "token_hash") {
		t.Fatalf("team state did not store token hash: %s", string(data))
	}
}
