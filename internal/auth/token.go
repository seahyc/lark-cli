package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/yjwong/lark-cli/internal/config"
	"github.com/yjwong/lark-cli/internal/scopes"
)

// TokenStore holds the OAuth tokens
type TokenStore struct {
	AccessToken           string    `json:"access_token"`
	RefreshToken          string    `json:"refresh_token"`
	ExpiresAt             time.Time `json:"expires_at"`
	RefreshTokenExpiresAt time.Time `json:"refresh_token_expires_at"`
	Scope                 string    `json:"scope"`
	UserID                string    `json:"user_id,omitempty"`
	mu                    sync.RWMutex
}

var (
	tokens           *TokenStore
	tokensOnce       sync.Once
	tenantTokens     *TenantTokenStore
	tenantTokensOnce sync.Once
)

// TenantTokenStore holds the tenant access token (app-level, not user-level)
type TenantTokenStore struct {
	AccessToken string    `json:"tenant_access_token"`
	ExpiresAt   time.Time `json:"expires_at"`
	mu          sync.RWMutex
}

// GetTokenStore returns the singleton token store
func GetTokenStore() *TokenStore {
	tokensOnce.Do(func() {
		tokens = &TokenStore{}
		tokens.Load()
	})
	return tokens
}

// Load reads tokens from disk
func (t *TokenStore) Load() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	path := config.TokensFilePath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No tokens yet, that's OK
		}
		return fmt.Errorf("failed to read tokens: %w", err)
	}

	if err := json.Unmarshal(data, t); err != nil {
		return fmt.Errorf("failed to parse tokens: %w", err)
	}

	return nil
}

// Save writes tokens to disk
func (t *TokenStore) Save() error {
	t.mu.RLock()
	defer t.mu.RUnlock()

	data, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal tokens: %w", err)
	}

	path := config.TokensFilePath()
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write tokens: %w", err)
	}

	return nil
}

// Clear removes all tokens
func (t *TokenStore) Clear() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.AccessToken = ""
	t.RefreshToken = ""
	t.ExpiresAt = time.Time{}
	t.RefreshTokenExpiresAt = time.Time{}
	t.Scope = ""
	t.UserID = ""

	path := config.TokensFilePath()
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove tokens file: %w", err)
	}

	return nil
}

// Update sets new token values and saves to disk
func (t *TokenStore) Update(accessToken, refreshToken string, expiresIn, refreshExpiresIn int, scope string) error {
	t.mu.Lock()
	t.AccessToken = accessToken
	t.RefreshToken = refreshToken
	t.ExpiresAt = time.Now().Add(time.Duration(expiresIn) * time.Second)
	if refreshExpiresIn > 0 {
		t.RefreshTokenExpiresAt = time.Now().Add(time.Duration(refreshExpiresIn) * time.Second)
	}
	t.Scope = scope
	t.mu.Unlock()

	return t.Save()
}

// IsValid checks if the access token is valid and not expired
func (t *TokenStore) IsValid() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.AccessToken == "" {
		return false
	}

	// Consider token invalid if it expires within 5 minutes
	return time.Now().Add(5 * time.Minute).Before(t.ExpiresAt)
}

// NeedsRefresh checks if the token should be refreshed
func (t *TokenStore) NeedsRefresh() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.AccessToken == "" {
		return false
	}

	// Refresh if expiring within 10 minutes
	return time.Now().Add(10 * time.Minute).After(t.ExpiresAt)
}

// CanRefresh checks if we have a valid refresh token
func (t *TokenStore) CanRefresh() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.RefreshToken == "" {
		return false
	}

	// Check if refresh token itself is expired
	if !t.RefreshTokenExpiresAt.IsZero() && time.Now().After(t.RefreshTokenExpiresAt) {
		return false
	}

	return true
}

// GetAccessToken returns the current access token
func (t *TokenStore) GetAccessToken() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.AccessToken
}

// GetRefreshToken returns the current refresh token
func (t *TokenStore) GetRefreshToken() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.RefreshToken
}

// GetExpiresAt returns when the access token expires
func (t *TokenStore) GetExpiresAt() time.Time {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.ExpiresAt
}

// GetScope returns the granted scope string
func (t *TokenStore) GetScope() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.Scope
}

// GetGrantedGroups returns a map of scope group names to whether they're fully granted
func (t *TokenStore) GetGrantedGroups() map[string]bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return scopes.GetGrantedGroups(t.Scope)
}

// GetGrantedGroupsList returns a list of fully granted scope group names
func (t *TokenStore) GetGrantedGroupsList() []string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return scopes.GetGrantedGroupsList(t.Scope)
}

// HasScope checks if a specific scope is granted
func (t *TokenStore) HasScope(scope string) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return scopes.CheckScope(scope, t.Scope)
}

// HasScopeGroup checks if all scopes for a group are granted
func (t *TokenStore) HasScopeGroup(groupName string) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	ok, _ := scopes.CheckScopeGroup(groupName, t.Scope)
	return ok
}

// --- Tenant Token Store ---

// GetTenantTokenStore returns the singleton tenant token store
func GetTenantTokenStore() *TenantTokenStore {
	tenantTokensOnce.Do(func() {
		tenantTokens = &TenantTokenStore{}
		tenantTokens.Load()
	})
	return tenantTokens
}

// Load reads tenant tokens from disk
func (t *TenantTokenStore) Load() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	path := config.TenantTokensFilePath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No tokens yet, that's OK
		}
		return fmt.Errorf("failed to read tenant tokens: %w", err)
	}

	if err := json.Unmarshal(data, t); err != nil {
		return fmt.Errorf("failed to parse tenant tokens: %w", err)
	}

	return nil
}

// Save writes tenant tokens to disk
func (t *TenantTokenStore) Save() error {
	t.mu.RLock()
	defer t.mu.RUnlock()

	data, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal tenant tokens: %w", err)
	}

	path := config.TenantTokensFilePath()
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write tenant tokens: %w", err)
	}

	return nil
}

// Update sets new tenant token values and saves to disk
func (t *TenantTokenStore) Update(accessToken string, expiresIn int) error {
	t.mu.Lock()
	t.AccessToken = accessToken
	t.ExpiresAt = time.Now().Add(time.Duration(expiresIn) * time.Second)
	t.mu.Unlock()

	return t.Save()
}

// IsValid checks if the tenant access token is valid and not expired
func (t *TenantTokenStore) IsValid() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.AccessToken == "" {
		return false
	}

	// Consider token invalid if it expires within 5 minutes
	return time.Now().Add(5 * time.Minute).Before(t.ExpiresAt)
}

// GetAccessToken returns the current tenant access token
func (t *TenantTokenStore) GetAccessToken() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.AccessToken
}
