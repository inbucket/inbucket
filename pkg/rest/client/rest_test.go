package client

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

const baseURLStr = "http://test.local:8080"
const baseURLPathStr = "http://test.local:8080/inbucket"

var baseURL *url.URL

var baseURLPath *url.URL

func init() {
	var err error
	baseURL, err = url.Parse(baseURLStr)
	if err != nil {
		panic(err)
	}
	baseURLPath, err = url.Parse(baseURLPathStr)
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

func TestDoTable(t *testing.T) {
	tests := []struct {
		method     string
		uri        string
		wantMethod string
		base       *url.URL
		wantURL    string
		wantBody   []byte
	}{
		{method: "GET", wantMethod: "GET", uri: "/doget", base: baseURL, wantURL: baseURLStr + "/doget", wantBody: []byte("Test body 1")},
		{method: "POST", wantMethod: "POST", uri: "/dopost", base: baseURL, wantURL: baseURLStr + "/dopost", wantBody: []byte("Test body 2")},
		{method: "GET", wantMethod: "GET", uri: "/doget", base: baseURLPath, wantURL: baseURLPathStr + "/doget", wantBody: []byte("Test body 3")},
		{method: "POST", wantMethod: "POST", uri: "/dopost", base: baseURLPath, wantURL: baseURLPathStr + "/dopost", wantBody: []byte("Test body 4")},
	}
	for _, test := range tests {
		testname := fmt.Sprintf("%s,%s", test.method, test.wantURL)
		t.Run(testname, func(t *testing.T) {
			ctx := context.Background()
			mth := &mockHTTPClient{}
			c := &restClient{mth, test.base}

			resp, err := c.do(ctx, test.method, test.uri, test.wantBody)
			require.NoError(t, err)
			err = resp.Body.Close()
			require.NoError(t, err)

			if mth.req.Method != test.wantMethod {
				t.Errorf("req.Method == %q, want %q", mth.req.Method, test.wantMethod)
			}
			if mth.req.URL.String() != test.wantURL {
				t.Errorf("req.URL == %q, want %q", mth.req.URL.String(), test.wantURL)
			}
			if !bytes.Equal(mth.ReqBody(), test.wantBody) {
				t.Errorf("req.Body == %q, want %q", mth.ReqBody(), test.wantBody)
			}
		})
	}
}

func TestDoJSON(t *testing.T) {
	var want, got string

	mth := &mockHTTPClient{
		body: `{"foo": "bar"}`,
	}
	c := &restClient{mth, baseURL}

	var v map[string]interface{}
	err := c.doJSON(context.Background(), "GET", "/doget", &v)
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

	err := c.doJSON(context.Background(), "GET", "/doget", nil)
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
