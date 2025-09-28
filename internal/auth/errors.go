package auth

import "github.com/Jayleonc/service/pkg/xerr"

var (
	ErrRefreshInvalidPayload = xerr.New(1001, "invalid request payload")
	ErrInvalidRefreshToken   = xerr.New(1002, "invalid refresh token")
	ErrRefreshFailed         = xerr.New(1003, "failed to refresh token")

	ErrMissingAuthorizationHeader = xerr.New(2001, "missing authorization header")
	ErrInvalidAuthorizationHeader = xerr.New(2002, "invalid authorization header")
	ErrInvalidToken               = xerr.New(2003, "invalid token")
)
