package middleware

import "github.com/Jayleonc/service/pkg/xerr"

var (
	ErrMissingSession        = xerr.New(2051, "missing session")
	ErrInsufficientPrivilege = xerr.New(2052, "insufficient permissions")
)
