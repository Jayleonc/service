package user

import "github.com/Jayleonc/service/pkg/xerr"

var (
	ErrRegisterInvalidPayload = xerr.New(3001, "invalid request payload")
	ErrRegisterFailed         = xerr.New(3002, "failed to register user")
	ErrEmailExists            = xerr.New(3003, "email already exists")

	ErrLoginInvalidPayload = xerr.New(3011, "invalid request payload")
	ErrLoginFailed         = xerr.New(3012, "failed to login user")
	ErrInvalidCredentials  = xerr.New(3013, "invalid credentials")

	ErrSessionMissing       = xerr.New(3021, "missing session")
	ErrProfileLookupFailed  = xerr.New(3022, "failed to load profile")
	ErrUpdateMeInvalidBody  = xerr.New(3032, "invalid request payload")
	ErrUpdateProfileFailed  = xerr.New(3033, "failed to update profile")
	ErrRolesRequired        = xerr.New(3034, "at least one role must be assigned")
	ErrCreateInvalidPayload = xerr.New(3041, "invalid request payload")
	ErrCreateFailed         = xerr.New(3042, "failed to create user")

	ErrUpdateInvalidPayload = xerr.New(3051, "invalid request payload")
	ErrInvalidUserID        = xerr.New(3052, "invalid user id")
	ErrUpdateUserFailed     = xerr.New(3053, "failed to update user")

	ErrDeleteInvalidPayload = xerr.New(3061, "invalid request payload")
	ErrDeleteInvalidUserID  = xerr.New(3062, "invalid user id")
	ErrDeleteUserFailed     = xerr.New(3063, "failed to delete user")

	ErrListInvalidPayload = xerr.New(3071, "invalid request payload")
	ErrListUsersFailed    = xerr.New(3072, "failed to list users")

	ErrAssignInvalidPayload = xerr.New(3081, "invalid request payload")
	ErrAssignInvalidUserID  = xerr.New(3082, "invalid user id")
	ErrAssignRolesFailed    = xerr.New(3083, "failed to assign roles")
)
