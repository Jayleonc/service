package middleware

import "github.com/Jayleonc/service/pkg/xerr"

// 中间件模块错误码范围：4000-4999
var (
	ErrMissingSession        = xerr.New(4001, "missing session")
	ErrInsufficientPrivilege = xerr.New(4002, "insufficient permissions")
)
