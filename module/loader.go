package module

import (
	"sandwich/config"
)

type ModuleLoader struct {
	mods []config.ModuleConfig
}

func NewModuleLoader() *ModuleLoader {
	return &ModuleLoader{}
}
