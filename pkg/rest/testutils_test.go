package rest

import (
	"bytes"
	"context"
	"log"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/inbucket/inbucket/v3/pkg/config"
	"github.com/inbucket/inbucket/v3/pkg/message"
	"github.com/inbucket/inbucket/v3/pkg/msghub"
	"github.com/inbucket/inbucket/v3/pkg/server/web"
)

func testRestGet(url string) (*httptest.ResponseRecorder, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	req.Header.Add("Accept", "application/json")
	if err != nil {
		return nil, err
	}

	// Pass request to handlers directly.
	w := httptest.NewRecorder()
	web.Router.ServeHTTP(w, req)

	return w, nil
}

func testRestPatch(url string, body string) (*httptest.ResponseRecorder, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, url, strings.NewReader(body))
	req.Header.Add("Accept", "application/json")
	if err != nil {
		return nil, err
	}

	// Pass request to handlers directly.
	w := httptest.NewRecorder()
	web.Router.ServeHTTP(w, req)

	return w, nil
}

func setupWebServer(mm message.Manager) *bytes.Buffer {
	// Capture log output
	buf := new(bytes.Buffer)
	log.SetOutput(buf)

	// Have to reset default mux to prevent duplicate routes
	cfg := &config.Root{
		Web: config.Web{
			UIDir: "../ui",
		},
	}
	SetupRoutes(web.Router.PathPrefix("/api/").Subrouter())
	web.NewServer(cfg, mm, &msghub.Hub{})

	return buf
}

func decodedBoolEquals(t *testing.T, json interface{}, path string, want bool) {
	t.Helper()
	els := strings.Split(path, "/")
	val, msg := getDecodedPath(json, els...)
	if msg != "" {
		t.Errorf("JSON result%s", msg)
		return
	}
	if got, ok := val.(bool); ok {
		if got == want {
			return
		}
	}
	t.Errorf("JSON result/%s == %v (%T), want: %v", path, val, val, want)
}

func decodedNumberEquals(t *testing.T, json interface{}, path string, want float64) {
	t.Helper()
	els := strings.Split(path, "/")
	val, msg := getDecodedPath(json, els...)
	if msg != "" {
		t.Errorf("JSON result%s", msg)
		return
	}
	got, ok := val.(float64)
	if ok {
		if got == want {
			return
		}
	}
	t.Errorf("JSON result/%s == %v (%T) %v (int64),\nwant: %v / %v",
		path, val, val, int64(got), want, int64(want))
}

func decodedStringEquals(t *testing.T, json interface{}, path string, want string) {
	t.Helper()
	els := strings.Split(path, "/")
	val, msg := getDecodedPath(json, els...)
	if msg != "" {
		t.Errorf("JSON result%s", msg)
		return
	}
	if got, ok := val.(string); ok {
		if got == want {
			return
		}
	}
	t.Errorf("JSON result/%s == %v (%T), want: %v", path, val, val, want)
}

// getDecodedPath recursively navigates the specified path, returing the requested element.  If
// something goes wrong, the returned string will contain an explanation.
//
// Named path elements require the parent element to be a map[string]interface{}, numbers in square
// brackets require the parent element to be a []interface{}.
//
//	getDecodedPath(o, "users", "[1]", "name")
//
// is equivalent to the JavaScript:
//
//	o.users[1].name
func getDecodedPath(o interface{}, path ...string) (interface{}, string) {
	if len(path) == 0 {
		return o, ""
	}
	if o == nil {
		return nil, " is nil"
	}

	var present bool
	var val interface{}
	key := path[0]
	if key[0] == '[' {
		// Expecting slice.
		index, err := strconv.Atoi(strings.Trim(key, "[]"))
		if err != nil {
			return nil, "/" + key + " is not a slice index"
		}
		oslice, ok := o.([]interface{})
		if !ok {
			return nil, " is not a slice"
		}
		if index >= len(oslice) {
			return nil, "/" + key + " is out of bounds"
		}
		val, present = oslice[index], true
	} else {
		// Expecting map.
		omap, ok := o.(map[string]interface{})
		if !ok {
			return nil, " is not a map"
		}
		val, present = omap[key]
	}

	if !present {
		return nil, "/" + key + " is missing"
	}

	result, msg := getDecodedPath(val, path[1:]...)
	if msg != "" {
		return nil, "/" + key + msg
	}

	return result, ""
}
