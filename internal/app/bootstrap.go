package app

import (
	"github.com/Jayleonc/service/internal/auth"
	"github.com/Jayleonc/service/internal/feature"
	"github.com/Jayleonc/service/internal/role"
	"github.com/Jayleonc/service/internal/server"
	"github.com/Jayleonc/service/internal/user"
)

// Modules enumerates all modules that should be initialised at startup.
var Modules = []feature.Entry{
	{Name: "auth", Registrar: auth.Register},
	{Name: "role", Registrar: role.Register},
	{Name: "user", Registrar: user.Register},
}

// Bootstrap assembles the shared infrastructure and registers every module.
func Bootstrap() (*server.App, error) {
	return server.Bootstrap(Modules)
}
