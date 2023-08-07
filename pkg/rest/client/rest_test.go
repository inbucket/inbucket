package client

import (
	"bytes"
	"io"
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
	if m.statusCode == 0 {
		m.statusCode = 200
	}
	resp = &http.Response{
		StatusCode: m.statusCode,
		Body:       io.NopCloser(bytes.NewBufferString(m.body)),
	}
	return
}

func (m *mockHTTPClient) ReqBody() []byte {
	r, err := m.req.GetBody()
	if err != nil {
		return nil
	}
	body, err := io.ReadAll(r)
	if err != nil {
		return nil
	}
	_ = r.Close()
	return body
}

func TestDo(t *testing.T) {
	var want, got string
	mth := &mockHTTPClient{}
	c := &restClient{mth, baseURL}
	body := []byte("Test body")

	_, err := c.do("POST", "/dopost", body)
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

	b := mth.ReqBody()
	if !bytes.Equal(b, body) {
		t.Errorf("req.Body == %q, want %q", b, body)
	}
}

func TestDoJSON(t *testing.T) {
	var want, got string

	mth := &mockHTTPClient{
		body: `{"foo": "bar"}`,
	}
	c := &restClient{mth, baseURL}

	var v map[string]interface{}
	err := c.doJSON("GET", "/doget", &v)
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

	mth := &mockHTTPClient{}
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
