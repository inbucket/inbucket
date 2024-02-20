package stringutil

import (
	"crypto/sha1"
	"encoding/hex"
	"io"
	"net/mail"
	"strings"
)

// HashMailboxName accepts a mailbox name and hashes it.  filestore uses this as
// the directory to house the mailbox
func HashMailboxName(mailbox string) string {
	h := sha1.New()
	if _, err := io.WriteString(h, mailbox); err != nil {
		// This should never happen.
		return ""
	}

	return hex.EncodeToString(h.Sum(nil))
}

// StringAddress converts an Address to a UTF-8 string.
func StringAddress(a *mail.Address) string {
	b := &strings.Builder{}
	if a != nil {
		if a.Name != "" {
			b.WriteString(a.Name)
			b.WriteRune(' ')
		}
		if a.Address != "" {
			b.WriteRune('<')
			b.WriteString(a.Address)
			b.WriteRune('>')
		}
	}
	return b.String()
}

// StringAddressList converts a list of addresses to a list of UTF-8 strings.
func StringAddressList(addrs []*mail.Address) []string {
	s := make([]string, len(addrs))
	for i, a := range addrs {
		s[i] = StringAddress(a)
	}
	return s
}

// SliceContains returns true if s is present in slice.
func SliceContains(slice []string, s string) bool {
	for _, v := range slice {
		if s == v {
			return true
		}
	}
	return false
}

// SliceToLower lowercases the contents of slice of strings.
func SliceToLower(slice []string) {
	for i, s := range slice {
		slice[i] = strings.ToLower(s)
	}
}

// MakePathPrefixer returns a function that will add the specified prefix (base) to URI strings.
// The returned prefixer expects all provided paths to start with /.
func MakePathPrefixer(prefix string) func(string) string {
	prefix = strings.Trim(prefix, "/")
	if prefix != "" {
		prefix = "/" + prefix
	}

	return func(path string) string {
		return prefix + path
	}
}

// MatchWithWildcards tests if a "s" string matches a "p" pattern with wildcards (*, ?)
func MatchWithWildcards(p string, s string) bool {
	runeInput := []rune(s)
	runePattern := []rune(p)
	lenInput := len(runeInput)
	lenPattern := len(runePattern)
	isMatchingMatrix := make([][]bool, lenInput+1)
	for i := range isMatchingMatrix {
		isMatchingMatrix[i] = make([]bool, lenPattern+1)
	}
	isMatchingMatrix[0][0] = true
	if lenPattern > 0 {
		if runePattern[0] == '*' {
			isMatchingMatrix[0][1] = true
		}
	}
	for j := 2; j <= lenPattern; j++ {
		if runePattern[j-1] == '*' {
			isMatchingMatrix[0][j] = isMatchingMatrix[0][j-1]
		}
	}
	for i := 1; i <= lenInput; i++ {
		for j := 1; j <= lenPattern; j++ {
			if runePattern[j-1] == '*' {
				isMatchingMatrix[i][j] = isMatchingMatrix[i-1][j] || isMatchingMatrix[i][j-1]
			}

			if runePattern[j-1] == '?' || runeInput[i-1] == runePattern[j-1] {
				isMatchingMatrix[i][j] = isMatchingMatrix[i-1][j-1]
			}
		}
	}
	return isMatchingMatrix[lenInput][lenPattern]
}
