package user

import "github.com/Jayleonc/service/pkg/xerr"

// 用户模块错误码范围：2000-2999
var (
	ErrRegisterFailed      = xerr.New(2001, "failed to register user")
	ErrEmailExists         = xerr.New(2002, "email already exists")
	ErrLoginFailed         = xerr.New(2011, "failed to login user")
	ErrInvalidCredentials  = xerr.New(2012, "invalid credentials")
	ErrProfileLookupFailed = xerr.New(2021, "failed to load profile")
	ErrUpdateProfileFailed = xerr.New(2022, "failed to update profile")
	ErrRolesRequired       = xerr.New(2023, "at least one role must be assigned")
	ErrCreateFailed        = xerr.New(2031, "failed to create user")
	ErrUpdateUserFailed    = xerr.New(2032, "failed to update user")
	ErrDeleteUserFailed    = xerr.New(2033, "failed to delete user")
	ErrListUsersFailed     = xerr.New(2034, "failed to list users")
	ErrAssignRolesFailed   = xerr.New(2035, "failed to assign roles")
)
