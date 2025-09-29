package rbac

// Resource 定义系统内可授权的资源标识。
const (
	ResourceUser           = "user"
	ResourceRBACRole       = "rbac.role"
	ResourceRBACPermission = "rbac.permission"
	ResourceSystem         = "system"
)

// Action 定义系统内可授权的操作标识。
const (
	ActionCreate            = "create"
	ActionRead              = "read"
	ActionUpdate            = "update"
	ActionDelete            = "delete"
	ActionList              = "list"
	ActionAssignRoles       = "assign_roles"
	ActionAssignPermissions = "assign_permissions"
	ActionViewPermissions   = "view_permissions"
	ActionAdmin             = "admin"
)

// PermissionKey 将资源与操作组合为权限键。
func PermissionKey(resource, action string) string {
	if resource == "" {
		return action
	}
	if action == "" {
		return resource
	}
	return resource + ":" + action
}
