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
	req *http.Request
}

func (m *mockHTTPClient) Do(req *http.Request) (resp *http.Response, err error) {
	m.req = req
	resp = &http.Response{
		Body: ioutil.NopCloser(&bytes.Buffer{}),
	}

	return
}

func TestDo(t *testing.T) {
	var want, got string

	mth := &mockHTTPClient{}
	c := &restClient{mth, baseURL}

	c.do("POST", "/dopost")

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

func TestDoGet(t *testing.T) {
	var want, got string

	mth := &mockHTTPClient{}
	c := &restClient{mth, baseURL}

	v := new(map[string]interface{})
	c.doGet("/doget", &v)

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
