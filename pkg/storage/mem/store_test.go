package mem

import (
	"testing"

	"github.com/jhillyerd/inbucket/pkg/storage"
	"github.com/jhillyerd/inbucket/pkg/test"
)

// TestSuite runs storage package test suite on file store.
func TestSuite(t *testing.T) {
	test.StoreSuite(t, func() (storage.Store, func(), error) {
		s := New()
		destroy := func() {}
		return s, destroy, nil
	})
}
