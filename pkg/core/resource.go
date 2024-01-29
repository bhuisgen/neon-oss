package core

import "time"

type Resource struct {
	Data [][]byte
	TTL  time.Duration
}
