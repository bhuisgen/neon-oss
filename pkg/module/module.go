package module

import (
	"fmt"
	"sync"
)

// Module is the interface of module.
type Module interface {
	// ModuleInfo returns the module information.
	ModuleInfo() ModuleInfo
}

// ModuleID is the module id.
type ModuleID string

// ModuleInfo implements the module information.
type ModuleInfo struct {
	// ID is the module ID.
	ID ModuleID
	// NewInstance returns a new module instance.
	NewInstance func() Module
}

var (
	modules     = make(map[ModuleID]ModuleInfo)
	modulesLock sync.RWMutex
)

// Register registers a module.
func Register(module Module) {
	info := module.ModuleInfo()
	modulesLock.Lock()
	if _, ok := modules[info.ID]; !ok {
		modules[info.ID] = info
	}
	modulesLock.Unlock()
}

// Unregister unregisters a module.
func Unregister(module Module) {
	modulesLock.Lock()
	delete(modules, module.ModuleInfo().ID)
	modulesLock.Unlock()
}

// Lookup returns the module information if found.
func Lookup(id ModuleID) (ModuleInfo, error) {
	modulesLock.RLock()
	mi, ok := modules[id]
	modulesLock.RUnlock()
	if !ok {
		return ModuleInfo{}, fmt.Errorf("module '%s' not registered", id)
	}
	return mi, nil
}
