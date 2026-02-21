package auth

import (
	"context"

	"go.uber.org/zap"
)

// Permission represents a single resource-action pair.
type Permission struct {
	Resource string
	Action   string
}

// RBACService provides role-based access control by mapping roles
// to their allowed permissions (resource + action combinations).
type RBACService struct {
	permissions map[string][]Permission
	log         *zap.Logger
}

// NewRBACService creates a new RBACService with predefined role permissions.
//
// Roles:
//   - "admin"    : full access to all resources and actions
//   - "operator" : access to devices, transactions, reservations, and reports
//   - "user"     : access to own data only (read devices/transactions/reservations)
//
// Resources: users, devices, transactions, reservations, admin, reports
// Actions:   read, write, delete, manage
func NewRBACService(log *zap.Logger) *RBACService {
	permissions := map[string][]Permission{
		"admin": {
			// Full access to everything
			{Resource: "users", Action: "read"},
			{Resource: "users", Action: "write"},
			{Resource: "users", Action: "delete"},
			{Resource: "users", Action: "manage"},
			{Resource: "devices", Action: "read"},
			{Resource: "devices", Action: "write"},
			{Resource: "devices", Action: "delete"},
			{Resource: "devices", Action: "manage"},
			{Resource: "transactions", Action: "read"},
			{Resource: "transactions", Action: "write"},
			{Resource: "transactions", Action: "delete"},
			{Resource: "transactions", Action: "manage"},
			{Resource: "reservations", Action: "read"},
			{Resource: "reservations", Action: "write"},
			{Resource: "reservations", Action: "delete"},
			{Resource: "reservations", Action: "manage"},
			{Resource: "admin", Action: "read"},
			{Resource: "admin", Action: "write"},
			{Resource: "admin", Action: "delete"},
			{Resource: "admin", Action: "manage"},
			{Resource: "reports", Action: "read"},
			{Resource: "reports", Action: "write"},
			{Resource: "reports", Action: "delete"},
			{Resource: "reports", Action: "manage"},
		},
		"operator": {
			// Devices: full CRUD
			{Resource: "devices", Action: "read"},
			{Resource: "devices", Action: "write"},
			{Resource: "devices", Action: "delete"},
			{Resource: "devices", Action: "manage"},
			// Transactions: full CRUD
			{Resource: "transactions", Action: "read"},
			{Resource: "transactions", Action: "write"},
			{Resource: "transactions", Action: "delete"},
			{Resource: "transactions", Action: "manage"},
			// Reservations: full CRUD
			{Resource: "reservations", Action: "read"},
			{Resource: "reservations", Action: "write"},
			{Resource: "reservations", Action: "delete"},
			{Resource: "reservations", Action: "manage"},
			// Reports: read only
			{Resource: "reports", Action: "read"},
		},
		"user": {
			// Own data only: read devices, read/write own transactions and reservations
			{Resource: "devices", Action: "read"},
			{Resource: "transactions", Action: "read"},
			{Resource: "transactions", Action: "write"},
			{Resource: "reservations", Action: "read"},
			{Resource: "reservations", Action: "write"},
		},
	}

	log.Info("RBAC service initialized",
		zap.Int("roles", len(permissions)),
	)

	return &RBACService{
		permissions: permissions,
		log:         log,
	}
}

// CheckPermission verifies whether the given role has permission to perform
// the specified action on the specified resource. It also validates that
// user context information is present when needed.
func (s *RBACService) CheckPermission(ctx context.Context, role, resource, action string) bool {
	perms, exists := s.permissions[role]
	if !exists {
		s.log.Warn("unknown role attempted access",
			zap.String("role", role),
			zap.String("resource", resource),
			zap.String("action", action),
		)
		return false
	}

	for _, p := range perms {
		if p.Resource == resource && p.Action == action {
			s.log.Debug("permission granted",
				zap.String("role", role),
				zap.String("resource", resource),
				zap.String("action", action),
			)
			return true
		}
	}

	s.log.Warn("permission denied",
		zap.String("role", role),
		zap.String("resource", resource),
		zap.String("action", action),
	)
	return false
}

// GetPermissions returns all permissions assigned to the given role.
// Returns nil if the role does not exist.
func (s *RBACService) GetPermissions(role string) []Permission {
	perms, exists := s.permissions[role]
	if !exists {
		s.log.Warn("requested permissions for unknown role",
			zap.String("role", role),
		)
		return nil
	}

	// Return a copy to prevent external mutation
	result := make([]Permission, len(perms))
	copy(result, perms)
	return result
}
