package server

import (
	authmodule "github.com/Jayleonc/service/internal/auth"
	"github.com/Jayleonc/service/internal/module"
	"github.com/Jayleonc/service/internal/role"
	usermodule "github.com/Jayleonc/service/internal/user"
)

// Modules enumerates all modules that should be initialised at startup.
var Modules = []module.Entry{
	{Name: "auth", Registrar: authmodule.Register},
	{Name: "user", Registrar: usermodule.Register},
	{Name: "role", Registrar: role.Register},
}
