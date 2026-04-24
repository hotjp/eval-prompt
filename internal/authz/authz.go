// Package authz implements L3-Authz layer: permission checks (RBAC/OpenFGA),
// rate limiting, and identity verification.
package authz

import (
	"context"
)

// Authz handles authorization and rate limiting.
type Authz struct {
	// TODO: Add dependencies
}

// New creates a new Authz instance.
func New() *Authz {
	return &Authz{}
}

// CheckPermission checks if the request has permission to access the resource.
func (a *Authz) CheckPermission(ctx context.Context, subject, action, resource string) error {
	// TODO: Implement permission check
	return nil
}

// ValidateToken validates the JWT token and returns claims.
func (a *Authz) ValidateToken(ctx context.Context, token string) (map[string]interface{}, error) {
	// TODO: Implement token validation
	return nil, nil
}
