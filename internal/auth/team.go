package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"moyuan-code/internal/fsutil"
	"moyuan-code/internal/logging"
	"moyuan-code/internal/textutil"
	"moyuan-code/internal/workspace"
)

type Session struct {
	ID           string   `json:"id"`
	UserID       string   `json:"user_id"`
	DisplayName  string   `json:"display_name,omitempty"`
	Roles        []string `json:"roles"`
	Status       string   `json:"status"`
	CreatedAt    string   `json:"created_at"`
	RevokedAt    string   `json:"revoked_at,omitempty"`
	RevokedBy    string   `json:"revoked_by,omitempty"`
	RevokeReason string   `json:"revoke_reason,omitempty"`
}

type APIToken struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	ActorID      string   `json:"actor_id"`
	Scopes       []string `json:"scopes"`
	TokenPrefix  string   `json:"token_prefix"`
	TokenHash    string   `json:"token_hash,omitempty"`
	Status       string   `json:"status"`
	CreatedAt    string   `json:"created_at"`
	RevokedAt    string   `json:"revoked_at,omitempty"`
	RevokedBy    string   `json:"revoked_by,omitempty"`
	RevokeReason string   `json:"revoke_reason,omitempty"`
}

type APITokenCreated struct {
	Token      APIToken `json:"token"`
	TokenValue string   `json:"token_value"`
}

type ServiceAccount struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Roles     []string `json:"roles"`
	Status    string   `json:"status"`
	CreatedAt string   `json:"created_at"`
}

type CreateSessionOptions struct {
	UserID      string   `json:"user_id"`
	DisplayName string   `json:"display_name,omitempty"`
	Roles       []string `json:"roles,omitempty"`
}

type CreateTokenOptions struct {
	Name    string   `json:"name"`
	ActorID string   `json:"actor_id"`
	Scopes  []string `json:"scopes,omitempty"`
}

type CreateServiceAccountOptions struct {
	ID    string   `json:"id,omitempty"`
	Name  string   `json:"name"`
	Roles []string `json:"roles,omitempty"`
}

type RevokeOptions struct {
	ActorID string `json:"actor_id,omitempty"`
	Reason  string `json:"reason,omitempty"`
}

type teamState struct {
	SchemaVersion   int              `json:"schema_version"`
	Sessions        []Session        `json:"sessions"`
	APITokens       []APIToken       `json:"api_tokens"`
	ServiceAccounts []ServiceAccount `json:"service_accounts"`
}

func CreateSession(rootDir string, options CreateSessionOptions) (Session, error) {
	state, err := loadTeamState(rootDir)
	if err != nil {
		return Session{}, err
	}
	userID := normalizeIdentity(options.UserID)
	if userID == "" {
		return Session{}, errors.New("session_user_id_required")
	}
	now := time.Now().UTC()
	session := Session{
		ID:          "session-" + textutil.Slugify(userID) + "-" + timeID(now),
		UserID:      userID,
		DisplayName: strings.TrimSpace(options.DisplayName),
		Roles:       normalizeList(options.Roles, []string{"developer"}),
		Status:      "active",
		CreatedAt:   now.Format(time.RFC3339Nano),
	}
	state.Sessions = append([]Session{session}, state.Sessions...)
	if err := saveTeamState(rootDir, state); err != nil {
		return Session{}, err
	}
	_ = logging.Log(rootDir, "audit", "auth.session.created", map[string]any{"session_id": session.ID, "user_id": session.UserID, "roles": session.Roles})
	return session, nil
}

func ListSessions(rootDir string) ([]Session, error) {
	state, err := loadTeamState(rootDir)
	if err != nil {
		return nil, err
	}
	sort.SliceStable(state.Sessions, func(i, j int) bool {
		return state.Sessions[i].CreatedAt > state.Sessions[j].CreatedAt
	})
	return state.Sessions, nil
}

func RevokeSession(rootDir string, sessionID string, options RevokeOptions) (Session, bool, error) {
	state, err := loadTeamState(rootDir)
	if err != nil {
		return Session{}, false, err
	}
	for i := range state.Sessions {
		if state.Sessions[i].ID != strings.TrimSpace(sessionID) {
			continue
		}
		if state.Sessions[i].Status != "revoked" {
			state.Sessions[i].Status = "revoked"
			state.Sessions[i].RevokedAt = time.Now().UTC().Format(time.RFC3339Nano)
			state.Sessions[i].RevokedBy = normalizeActor(options.ActorID)
			state.Sessions[i].RevokeReason = strings.TrimSpace(options.Reason)
		}
		if err := saveTeamState(rootDir, state); err != nil {
			return Session{}, true, err
		}
		_ = logging.Log(rootDir, "audit", "auth.session.revoked", map[string]any{"session_id": state.Sessions[i].ID, "revoked_by": state.Sessions[i].RevokedBy})
		return state.Sessions[i], true, nil
	}
	return Session{}, false, nil
}

func CreateAPIToken(rootDir string, options CreateTokenOptions) (APITokenCreated, error) {
	state, err := loadTeamState(rootDir)
	if err != nil {
		return APITokenCreated{}, err
	}
	name := strings.TrimSpace(options.Name)
	if name == "" {
		return APITokenCreated{}, errors.New("api_token_name_required")
	}
	actorID := normalizeIdentity(options.ActorID)
	if actorID == "" {
		return APITokenCreated{}, errors.New("api_token_actor_id_required")
	}
	rawToken, err := randomToken()
	if err != nil {
		return APITokenCreated{}, err
	}
	now := time.Now().UTC()
	token := APIToken{
		ID:          "api-token-" + textutil.Slugify(name) + "-" + timeID(now),
		Name:        name,
		ActorID:     actorID,
		Scopes:      normalizeList(options.Scopes, []string{"project:read"}),
		TokenPrefix: rawToken[:18],
		TokenHash:   hashToken(rawToken),
		Status:      "active",
		CreatedAt:   now.Format(time.RFC3339Nano),
	}
	state.APITokens = append([]APIToken{token}, state.APITokens...)
	if err := saveTeamState(rootDir, state); err != nil {
		return APITokenCreated{}, err
	}
	_ = logging.Log(rootDir, "audit", "auth.token.created", map[string]any{"api_token_id": token.ID, "actor_id": token.ActorID, "scopes": token.Scopes, "token_prefix": token.TokenPrefix})
	return APITokenCreated{Token: publicToken(token), TokenValue: rawToken}, nil
}

func ListAPITokens(rootDir string) ([]APIToken, error) {
	state, err := loadTeamState(rootDir)
	if err != nil {
		return nil, err
	}
	tokens := make([]APIToken, 0, len(state.APITokens))
	for _, token := range state.APITokens {
		tokens = append(tokens, publicToken(token))
	}
	sort.SliceStable(tokens, func(i, j int) bool {
		return tokens[i].CreatedAt > tokens[j].CreatedAt
	})
	return tokens, nil
}

func RevokeAPIToken(rootDir string, tokenID string, options RevokeOptions) (APIToken, bool, error) {
	state, err := loadTeamState(rootDir)
	if err != nil {
		return APIToken{}, false, err
	}
	for i := range state.APITokens {
		if state.APITokens[i].ID != strings.TrimSpace(tokenID) {
			continue
		}
		if state.APITokens[i].Status != "revoked" {
			state.APITokens[i].Status = "revoked"
			state.APITokens[i].RevokedAt = time.Now().UTC().Format(time.RFC3339Nano)
			state.APITokens[i].RevokedBy = normalizeActor(options.ActorID)
			state.APITokens[i].RevokeReason = strings.TrimSpace(options.Reason)
		}
		if err := saveTeamState(rootDir, state); err != nil {
			return APIToken{}, true, err
		}
		_ = logging.Log(rootDir, "audit", "auth.token.revoked", map[string]any{"api_token_id": state.APITokens[i].ID, "revoked_by": state.APITokens[i].RevokedBy})
		return publicToken(state.APITokens[i]), true, nil
	}
	return APIToken{}, false, nil
}

func CreateServiceAccount(rootDir string, options CreateServiceAccountOptions) (ServiceAccount, error) {
	state, err := loadTeamState(rootDir)
	if err != nil {
		return ServiceAccount{}, err
	}
	name := strings.TrimSpace(options.Name)
	if name == "" {
		return ServiceAccount{}, errors.New("service_account_name_required")
	}
	id := normalizeIdentity(options.ID)
	if id == "" {
		id = "svc-" + textutil.Slugify(name)
	}
	account := ServiceAccount{
		ID:        id,
		Name:      name,
		Roles:     normalizeList(options.Roles, []string{"service_account"}),
		Status:    "active",
		CreatedAt: time.Now().UTC().Format(time.RFC3339Nano),
	}
	next := []ServiceAccount{account}
	for _, existing := range state.ServiceAccounts {
		if existing.ID != account.ID {
			next = append(next, existing)
		}
	}
	state.ServiceAccounts = next
	if err := saveTeamState(rootDir, state); err != nil {
		return ServiceAccount{}, err
	}
	_ = logging.Log(rootDir, "audit", "auth.service_account.upserted", map[string]any{"service_account_id": account.ID, "roles": account.Roles})
	return account, nil
}

func ListServiceAccounts(rootDir string) ([]ServiceAccount, error) {
	state, err := loadTeamState(rootDir)
	if err != nil {
		return nil, err
	}
	sort.SliceStable(state.ServiceAccounts, func(i, j int) bool {
		return state.ServiceAccounts[i].CreatedAt > state.ServiceAccounts[j].CreatedAt
	})
	return state.ServiceAccounts, nil
}

func teamStatePath(rootDir string) string {
	return filepath.Join(workspace.ForRoot(rootDir).AuthDir, "team.json")
}

func loadTeamState(rootDir string) (teamState, error) {
	state := teamState{SchemaVersion: 1, Sessions: []Session{}, APITokens: []APIToken{}, ServiceAccounts: []ServiceAccount{}}
	_, err := fsutil.ReadJSON(teamStatePath(rootDir), &state)
	if state.SchemaVersion == 0 {
		state.SchemaVersion = 1
	}
	if state.Sessions == nil {
		state.Sessions = []Session{}
	}
	if state.APITokens == nil {
		state.APITokens = []APIToken{}
	}
	if state.ServiceAccounts == nil {
		state.ServiceAccounts = []ServiceAccount{}
	}
	return state, err
}

func saveTeamState(rootDir string, state teamState) error {
	if state.SchemaVersion == 0 {
		state.SchemaVersion = 1
	}
	return fsutil.WriteJSON(teamStatePath(rootDir), state)
}

func randomToken() (string, error) {
	data := make([]byte, 24)
	if _, err := rand.Read(data); err != nil {
		return "", err
	}
	return "moyuan_" + hex.EncodeToString(data), nil
}

func hashToken(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func publicToken(token APIToken) APIToken {
	token.TokenHash = ""
	return token
}

func timeID(value time.Time) string {
	return value.Format("20060102150405") + "-" + value.Format("000000000")
}

func normalizeIdentity(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	value = strings.ReplaceAll(value, " ", "-")
	return value
}

func normalizeActor(value string) string {
	value = normalizeIdentity(value)
	if value == "" {
		return "system"
	}
	return value
}

func normalizeList(values []string, fallback []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, value := range values {
		value = normalizeIdentity(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	if len(out) == 0 {
		return fallback
	}
	sort.Strings(out)
	return out
}
