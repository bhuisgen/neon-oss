package module

import (
	"fmt"
	"log"
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
	modulesLock.Lock()
	defer modulesLock.Unlock()

	info := module.ModuleInfo()
	if _, ok := modules[info.ID]; ok {
		log.Fatalf("Module '%s' already registered", info.ID)
	}
	modules[info.ID] = info
}

// Unregister unregisters a module.
func Unregister(module Module) {
	modulesLock.Lock()
	defer modulesLock.Unlock()

	delete(modules, module.ModuleInfo().ID)
}

// Lookup returns the module information if found.
func Lookup(id ModuleID) (ModuleInfo, error) {
	modulesLock.RLock()
	defer modulesLock.RUnlock()

	mi, ok := modules[id]
	if !ok {
		return ModuleInfo{}, fmt.Errorf("module '%s' not registered", id)
	}

	return mi, nil
}
