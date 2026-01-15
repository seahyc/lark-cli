package scopes

import "strings"

// CheckScope verifies if a specific scope is present in the granted scopes string
func CheckScope(required string, granted string) bool {
	grantedScopes := strings.Split(granted, " ")
	for _, s := range grantedScopes {
		if s == required {
			return true
		}
	}
	return false
}

// CheckScopeGroup verifies if all scopes for a group are granted
// Returns whether the group is satisfied and a list of missing scopes
func CheckScopeGroup(groupName string, granted string) (bool, []string) {
	group, ok := Groups[groupName]
	if !ok {
		return false, nil
	}

	missing := make([]string, 0)
	for _, scope := range group.Scopes {
		if !CheckScope(scope, granted) {
			missing = append(missing, scope)
		}
	}

	return len(missing) == 0, missing
}

// GetGrantedGroups returns a map of group names to whether they're fully granted
func GetGrantedGroups(granted string) map[string]bool {
	result := make(map[string]bool)
	for name := range Groups {
		ok, _ := CheckScopeGroup(name, granted)
		result[name] = ok
	}
	return result
}

// GetGrantedGroupsList returns a list of fully granted group names
func GetGrantedGroupsList(granted string) []string {
	var result []string
	for _, name := range AllGroupNames() {
		ok, _ := CheckScopeGroup(name, granted)
		if ok {
			result = append(result, name)
		}
	}
	return result
}

// ValidationError represents a scope validation failure
type ValidationError struct {
	Group   string
	Missing []string
}

// Error implements the error interface
func (e *ValidationError) Error() string {
	return "missing required scopes for " + e.Group
}

// ValidateForGroup checks if the granted scopes satisfy a group's requirements
// Returns nil if satisfied, or a ValidationError if not
func ValidateForGroup(groupName string, granted string) error {
	ok, missing := CheckScopeGroup(groupName, granted)
	if ok {
		return nil
	}
	return &ValidationError{
		Group:   groupName,
		Missing: missing,
	}
}
