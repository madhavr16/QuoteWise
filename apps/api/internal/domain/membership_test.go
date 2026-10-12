package domain

import "testing"

func TestValidRole(t *testing.T) {
	for _, role := range []Role{RoleOwner, RoleMember} {
		if !ValidRole(role) {
			t.Errorf("ValidRole(%q) = false, want true", role)
		}
	}
	for _, role := range []Role{"", "admin", "Owner", "member "} {
		if ValidRole(role) {
			t.Errorf("ValidRole(%q) = true, want false", role)
		}
	}
}
