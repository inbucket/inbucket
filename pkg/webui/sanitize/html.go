package sanitize

import (
	"bufio"
	"bytes"
	"io"
	"regexp"
	"strings"

	"github.com/microcosm-cc/bluemonday"
	"golang.org/x/net/html"
)

var (
	cssSafe = regexp.MustCompile(".*")
	policy  = bluemonday.UGCPolicy().
		AllowElements("center").
		AllowAttrs("style").Matching(cssSafe).Globally()
)

// HTML sanitizes the provided html, while attempting to preserve inline CSS styling.
func HTML(html string) (output string, err error) {
	output, err = sanitizeStyleTags(html)
	if err != nil {
		return "", err
	}
	output = policy.Sanitize(output)
	return
}

func sanitizeStyleTags(input string) (string, error) {
	r := strings.NewReader(input)
	b := &bytes.Buffer{}
	if err := styleTagFilter(b, r); err != nil {
		return "", err
	}
	return b.String(), nil
}

func styleTagFilter(w io.Writer, r io.Reader) error {
	bw := bufio.NewWriter(w)
	b := make([]byte, 0, 256)
	z := html.NewTokenizer(r)
	for {
		b = b[:0]
		tt := z.Next()
		switch tt {
		case html.ErrorToken:
			err := z.Err()
			if err == io.EOF {
				return bw.Flush()
			}
			return err
		case html.StartTagToken, html.SelfClosingTagToken:
			name, hasAttr := z.TagName()
			if !hasAttr {
				if _, err := bw.Write(z.Raw()); err != nil {
					return err
				}
				continue
			}
			b = append(b, '<')
			b = append(b, name...)
			for {
				key, val, more := z.TagAttr()
				strval := string(val)
				style := false
				if strings.ToLower(string(key)) == "style" {
					style = true
					strval = sanitizeStyle(strval)
				}
				if !style || strval != "" {
					b = append(b, ' ')
					b = append(b, key...)
					b = append(b, '=', '"')
					b = append(b, []byte(html.EscapeString(strval))...)
					b = append(b, '"')
				}
				if !more {
					break
				}
			}
			if tt == html.SelfClosingTagToken {
				b = append(b, '/')
			}
			if _, err := bw.Write(append(b, '>')); err != nil {
				return err
			}
		default:
			if _, err := bw.Write(z.Raw()); err != nil {
				return err
			}
		}
	}
}
