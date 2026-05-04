package role

import "github.com/healthcare-market-research/backend/internal/domain/user"

// Permission represents a specific permission
type Permission struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// RoleInfo contains detailed information about a role
type RoleInfo struct {
	Name        string       `json:"name"`
	DisplayName string       `json:"display_name"`
	Description string       `json:"description"`
	Permissions []Permission `json:"permissions"`
	Level       int          `json:"level"` // Hierarchy level (higher = more permissions)
}

// Permission definitions
var (
	// Report permissions
	PermViewReports = Permission{
		Name:        "reports.view",
		Description: "View published reports",
	}
	PermCreateReports = Permission{
		Name:        "reports.create",
		Description: "Create new reports",
	}
	PermEditReports = Permission{
		Name:        "reports.edit",
		Description: "Edit existing reports",
	}
	PermDeleteReports = Permission{
		Name:        "reports.delete",
		Description: "Delete reports",
	}
	PermPublishReports = Permission{
		Name:        "reports.publish",
		Description: "Publish reports",
	}

	// User permissions
	PermManageUsers = Permission{
		Name:        "users.manage",
		Description: "Create, edit, and delete users",
	}
	PermViewUsers = Permission{
		Name:        "users.view",
		Description: "View all users",
	}

	// Category permissions
	PermManageCategories = Permission{
		Name:        "categories.manage",
		Description: "Create, edit, and delete categories",
	}
	PermViewCategories = Permission{
		Name:        "categories.view",
		Description: "View all categories",
	}

	// Author permissions
	PermManageAuthors = Permission{
		Name:        "authors.manage",
		Description: "Create, edit, and delete authors",
	}
	PermViewAuthors = Permission{
		Name:        "authors.view",
		Description: "View all authors",
	}

	// Audit permissions
	PermViewAuditLogs = Permission{
		Name:        "audit.view",
		Description: "View audit logs",
	}
)

// Roles map - defines all available roles with their permissions
var Roles = map[string]RoleInfo{
	user.RoleViewer: {
		Name:        user.RoleViewer,
		DisplayName: "Viewer",
		Description: "Read-only access to published content",
		Level:       1,
		Permissions: []Permission{
			PermViewReports,
			PermViewCategories,
			PermViewAuthors,
		},
	},
	user.RoleEditor: {
		Name:        user.RoleEditor,
		DisplayName: "Editor",
		Description: "Can create and edit content",
		Level:       2,
		Permissions: []Permission{
			PermViewReports,
			PermCreateReports,
			PermEditReports,
			PermPublishReports,
			PermViewCategories,
			PermManageCategories,
			PermViewAuthors,
			PermManageAuthors,
		},
	},
	user.RoleAdmin: {
		Name:        user.RoleAdmin,
		DisplayName: "Administrator",
		Description: "Full system access including user management",
		Level:       3,
		Permissions: []Permission{
			PermViewReports,
			PermCreateReports,
			PermEditReports,
			PermDeleteReports,
			PermPublishReports,
			PermViewCategories,
			PermManageCategories,
			PermViewAuthors,
			PermManageAuthors,
			PermManageUsers,
			PermViewUsers,
			PermViewAuditLogs,
		},
	},
}

// GetRoleInfo returns information for a specific role
func GetRoleInfo(roleName string) (*RoleInfo, bool) {
	role, exists := Roles[roleName]
	return &role, exists
}

// GetAllRoles returns all available roles
func GetAllRoles() []RoleInfo {
	roles := make([]RoleInfo, 0, len(Roles))
	for _, role := range Roles {
		roles = append(roles, role)
	}
	return roles
}

// HasPermission checks if a role has a specific permission
func HasPermission(roleName string, permissionName string) bool {
	role, exists := Roles[roleName]
	if !exists {
		return false
	}

	for _, perm := range role.Permissions {
		if perm.Name == permissionName {
			return true
		}
	}
	return false
}
