package core

import "time"

// Resource implements a resource.
type Resource struct {
	// The data array.
	Data [][]byte
	// The time-to-leave.
	TTL time.Duration
}
