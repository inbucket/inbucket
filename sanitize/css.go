package sanitize

import (
	"bytes"
	"strings"

	"github.com/gorilla/css/scanner"
)

// propertyRule may someday allow control of what values are valid for a particular property.
type propertyRule struct{}

var allowedProperties = map[string]propertyRule{
	"align":            {},
	"background-color": {},
	"border":           {},
	"border-bottom":    {},
	"border-left":      {},
	"border-radius":    {},
	"border-right":     {},
	"border-top":       {},
	"box-sizing":       {},
	"clear":            {},
	"color":            {},
	"content":          {},
	"display":          {},
	"font-family":      {},
	"font-size":        {},
	"font-weight":      {},
	"height":           {},
	"line-height":      {},
	"margin":           {},
	"margin-bottom":    {},
	"margin-left":      {},
	"margin-right":     {},
	"margin-top":       {},
	"max-height":       {},
	"max-width":        {},
	"overflow":         {},
	"padding":          {},
	"padding-bottom":   {},
	"padding-left":     {},
	"padding-right":    {},
	"padding-top":      {},
	"table-layout":     {},
	"text-align":       {},
	"text-decoration":  {},
	"text-shadow":      {},
	"vertical-align":   {},
	"width":            {},
	"word-break":       {},
}

// Handler Token, return next state.
type stateHandler func(b *bytes.Buffer, t *scanner.Token) stateHandler

func sanitizeStyle(input string) string {
	b := &bytes.Buffer{}
	scan := scanner.New(input)
	state := stateStart
	for {
		t := scan.Next()
		if t.Type == scanner.TokenEOF {
			return b.String()
		}
		if t.Type == scanner.TokenError {
			return ""
		}
		state = state(b, t)
		if state == nil {
			return ""
		}
	}
}

func stateStart(b *bytes.Buffer, t *scanner.Token) stateHandler {
	switch t.Type {
	case scanner.TokenIdent:
		_, ok := allowedProperties[strings.ToLower(t.Value)]
		if !ok {
			return stateEat
		}
		b.WriteString(t.Value)
		return stateValid
	case scanner.TokenS:
		return stateStart
	}
	// Unexpected type.
	b.WriteString("/*" + t.Type.String() + "*/")
	return stateEat
}

func stateEat(b *bytes.Buffer, t *scanner.Token) stateHandler {
	if t.Type == scanner.TokenChar && t.Value == ";" {
		// Done eating.
		return stateStart
	}
	// Throw away this token.
	return stateEat
}

func stateValid(b *bytes.Buffer, t *scanner.Token) stateHandler {
	state := stateValid
	if t.Type == scanner.TokenChar && t.Value == ";" {
		// End of property.
		state = stateStart
	}
	b.WriteString(t.Value)
	return state
}
