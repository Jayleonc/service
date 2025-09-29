package rbac

import "github.com/Jayleonc/service/pkg/xerr"

// RBAC 模块错误码范围：3000-3999
var (
	ErrResourceNotFound = xerr.New(3001, "resource not found")
	ErrPermissionDenied = xerr.New(3002, "permission denied")
)
