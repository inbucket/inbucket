package sanitize

import (
	"testing"
)

func TestSanitizeStyle(t *testing.T) {
	testCases := []struct {
		input, want string
	}{
		{"", ""},
		{
			"color: red;",
			"color: red;",
		},
		{
			"background-color: black; color: white",
			"background-color: black;color: white",
		},
		{
			"background-color: black; invalid: true; color: white",
			"background-color: black;color: white",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			got := sanitizeStyle(tc.input)
			if got != tc.want {
				t.Errorf("got: %q, want: %q, input: %q", got, tc.want, tc.input)
			}
		})
	}
}
