package sanitize

import "github.com/microcosm-cc/bluemonday"

func HTML(html string) (output string, err error) {
	policy := bluemonday.UGCPolicy()
	output = policy.Sanitize(html)
	return
}
