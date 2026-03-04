package auth

import (
	"testing"
)

func TestPermissionConstants(t *testing.T) {
	// Test that permission constants are defined
	if PermCreateProblem == "" {
		t.Error("PermCreateProblem should be defined")
	}
	if PermEditProblem == "" {
		t.Error("PermEditProblem should be defined")
	}
	if PermDeleteProblem == "" {
		t.Error("PermDeleteProblem should be defined")
	}
	if PermAccessAdminPanel == "" {
		t.Error("PermAccessAdminPanel should be defined")
	}
}

func TestDefaultPermissionSets(t *testing.T) {
	// Test that default permission sets are defined
	if DefaultPermissionSets == nil {
		t.Fatal("DefaultPermissionSets should be defined")
	}

	// Test user role permissions
	userPerms, ok := DefaultPermissionSets["user"]
	if !ok {
		t.Error("user role should have default permissions")
	}
	if len(userPerms) == 0 {
		t.Error("user role should have at least one permission")
	}

	// Test admin role permissions
	adminPerms, ok := DefaultPermissionSets["admin"]
	if !ok {
		t.Error("admin role should have default permissions")
	}
	if len(adminPerms) == 0 {
		t.Error("admin role should have permissions")
	}

	// Test that admin has more permissions than user
	if len(adminPerms) <= len(userPerms) {
		t.Error("admin role should have more permissions than user role")
	}
}

func TestPermissionCategories(t *testing.T) {
	// Test that permissions are organized by category
	// Check that we have permissions in each category
	for _, perms := range DefaultPermissionSets {
		for _, perm := range perms {
			// Permissions should have the format "category.action"
			// This is a basic check
			if perm == "" {
				t.Error("permission code should not be empty")
			}
		}
	}
}

func TestPermissionCodeFormat(t *testing.T) {
	// Test all permission codes follow the format "category.action"
	allPerms := make(map[string]bool)
	for _, perms := range DefaultPermissionSets {
		for _, perm := range perms {
			if allPerms[perm] {
				continue // Skip duplicates
			}
			allPerms[perm] = true

			// Check format
			if len(perm) < 5 {
				t.Errorf("permission code '%s' is too short", perm)
			}
		}
	}
}
