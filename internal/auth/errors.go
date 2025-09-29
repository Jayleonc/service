package auth

import "github.com/Jayleonc/service/pkg/xerr"

// 认证模块错误码范围：1000-1999
var (
	ErrInvalidRefreshToken        = xerr.New(1001, "invalid refresh token")
	ErrRefreshFailed              = xerr.New(1002, "failed to refresh token")
	ErrMissingAuthorizationHeader = xerr.New(1101, "missing authorization header")
	ErrInvalidAuthorizationHeader = xerr.New(1102, "invalid authorization header")
	ErrInvalidToken               = xerr.New(1103, "invalid token")
)
