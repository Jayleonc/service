package rbac

import "strings"

// ParsePermissionKey splits a composite permission key into resource and action parts.
func ParsePermissionKey(key string) (resource, action string, ok bool) {
	trimmed := strings.TrimSpace(key)
	if trimmed == "" {
		return "", "", false
	}

	parts := strings.SplitN(trimmed, ":", 2)
	if len(parts) == 1 {
		return strings.ToLower(strings.TrimSpace(parts[0])), "", true
	}
	resource = strings.ToLower(strings.TrimSpace(parts[0]))
	action = strings.ToLower(strings.TrimSpace(parts[1]))
	if resource == "" && action == "" {
		return "", "", false
	}
	return resource, action, true
}

// NormalizeRoleName uppercases and trims a role identifier.
func NormalizeRoleName(name string) string {
	return strings.ToUpper(strings.TrimSpace(name))
}

// UniqueNormalized returns the deduplicated, normalized role names.
func UniqueNormalized(names []string) []string {
	if len(names) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(names))
	normalized := make([]string, 0, len(names))
	for _, name := range names {
		trimmed := NormalizeRoleName(name)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		normalized = append(normalized, trimmed)
	}
	return normalized
}
