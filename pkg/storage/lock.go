package storage

import (
	"strconv"
	"sync"
)

// HashLock holds a fixed length array of mutexes.  This approach allows concurrent mailbox
// access in most cases without requiring an infinite number of mutexes.
type HashLock [4096]sync.RWMutex

// Get returns a RWMutex based on the first 12 bits of the mailbox hash.  Hash must be a hexadecimal
// string of three or more characters.
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
