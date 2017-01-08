package client

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/url"
	"testing"
)

const baseURLStr = "http://test.local:8080"

var baseURL *url.URL

func init() {
	var err error
	baseURL, err = url.Parse(baseURLStr)
	if err != nil {
		panic(err)
	}
}

type mockHTTPClient struct {
	req        *http.Request
	statusCode int
	body       string
}

func (m *mockHTTPClient) Do(req *http.Request) (resp *http.Response, err error) {
	m.req = req
	resp = &http.Response{
		StatusCode: m.statusCode,
		Body:       ioutil.NopCloser(bytes.NewBufferString(m.body)),
	}

	return
}

func TestDo(t *testing.T) {
	var want, got string

	mth := &mockHTTPClient{}
	c := &restClient{mth, baseURL}

	_, err := c.do("POST", "/dopost")
	if err != nil {
		t.Fatal(err)
	}

	want = "POST"
	got = mth.req.Method
	if got != want {
		t.Errorf("req.Method == %q, want %q", got, want)
	}

	want = baseURLStr + "/dopost"
	got = mth.req.URL.String()
	if got != want {
		t.Errorf("req.URL == %q, want %q", got, want)
	}
}

func TestDoJSON(t *testing.T) {
	var want, got string

	mth := &mockHTTPClient{
		statusCode: 200,
		body:       `{"foo": "bar"}`,
	}
	c := &restClient{mth, baseURL}

	var v map[string]interface{}
	c.doJSON("GET", "/doget", &v)

	want = "GET"
	got = mth.req.Method
	if got != want {
		t.Errorf("req.Method == %q, want %q", got, want)
	}

	want = baseURLStr + "/doget"
	got = mth.req.URL.String()
	if got != want {
		t.Errorf("req.URL == %q, want %q", got, want)
	}

	want = "bar"
	if val, ok := v["foo"]; ok {
		got = val.(string)
		if got != want {
			t.Errorf("map[foo] == %q, want: %q", got, want)
		}
	} else {
		t.Errorf("Map did not contain key foo, want: %q", want)
	}
}

func TestDoJSONNilV(t *testing.T) {
	var want, got string

	mth := &mockHTTPClient{statusCode: 200}
	c := &restClient{mth, baseURL}

	err := c.doJSON("GET", "/doget", nil)
	if err != nil {
		t.Fatal(err)
	}

	want = "GET"
	got = mth.req.Method
	if got != want {
		t.Errorf("req.Method == %q, want %q", got, want)
	}

	want = baseURLStr + "/doget"
	got = mth.req.URL.String()
	if got != want {
		t.Errorf("req.URL == %q, want %q", got, want)
	}
}
