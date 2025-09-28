package role

import "github.com/Jayleonc/service/pkg/xerr"

var (
	ErrCreateInvalidPayload      = xerr.New(4001, "invalid request payload")
	ErrCreateDatabaseUnavailable = xerr.New(4003, "database not initialised")
	ErrRoleNameRequired          = xerr.New(4004, "role name is required")
	ErrCreateFailed              = xerr.New(4005, "failed to create role")

	ErrUpdateInvalidPayload      = xerr.New(4011, "invalid request payload")
	ErrInvalidRoleID             = xerr.New(4012, "invalid role id")
	ErrUpdateDatabaseUnavailable = xerr.New(4014, "database not initialised")
	ErrRoleNotFound              = xerr.New(4015, "role not found")
	ErrLoadRoleFailed            = xerr.New(4016, "failed to load role")
	ErrUpdateRoleNameEmpty       = xerr.New(4017, "role name is required")
	ErrUpdateFailed              = xerr.New(4018, "failed to update role")

	ErrDeleteInvalidPayload      = xerr.New(4021, "invalid request payload")
	ErrDeleteInvalidRoleID       = xerr.New(4022, "invalid role id")
	ErrDeleteFailed              = xerr.New(4023, "failed to delete role")
	ErrDeleteDatabaseUnavailable = xerr.New(4024, "database not initialised")

	ErrListDatabaseUnavailable = xerr.New(4032, "database not initialised")
	ErrListFailed              = xerr.New(4033, "failed to list roles")
)
