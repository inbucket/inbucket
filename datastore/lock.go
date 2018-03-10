package datastore

import (
	"strconv"
	"sync"
)

type HashLock [4096]sync.RWMutex

func (h *HashLock) Get(hash string) *sync.RWMutex {
	if len(hash) < 3 {
		return nil
	}
	i, err := strconv.ParseInt(hash[0:3], 16, 0)
	if err != nil {
		return nil
	}
	return &h[i]
}
