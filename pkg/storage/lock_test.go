package storage_test

import (
	"testing"

	"github.com/inbucket/inbucket/v3/pkg/storage"
)

func TestHashLock(t *testing.T) {
	hl := &storage.HashLock{}

	// Invalid hashes
	testCases := []struct {
		name, input string
	}{
		{"empty", ""},
		{"short", "a0"},
		{"badhex", "zzzzzzzzzzzzzzzzzzzzzzz"},
	}
	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			l := hl.Get(tc.input)
			if l != nil {
				t.Errorf("Expected nil lock for %s %q, got %v", tc.name, tc.input, l)
			}
		})
	}

	// Valid hashes
	testStrings := []string{
		"deadbeef",
		"00000000",
		"ffffffff",
	}
	for _, ts := range testStrings {
		t.Run(ts, func(t *testing.T) {
			l := hl.Get(ts)
			if l == nil {
				t.Errorf("Expected non-nil lock for hex string %q", ts)
			}
		})
	}

	a := hl.Get("deadbeef")
	b := hl.Get("deadbeef")
	if a != b {
		t.Errorf("Expected identical locks for identical hashes, got: %p != %p", a, b)
	}

	a = hl.Get("deadbeef")
	b = hl.Get("d3adb33f")
	if a == b {
		t.Errorf("Expected different locks for different hashes, got: %p == %p", a, b)
	}

	a = hl.Get("deadbeef")
	b = hl.Get("deadb33f")
	if a != b {
		t.Errorf("Expected identical locks for identical leading hashes, got: %p != %p", a, b)
	}
}
