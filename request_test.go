package intercept

import (
	"bytes"
	"errors"
	"github.com/nbio/st"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
)

var errRead = errors.New("read error")

type errorReader struct {
}

func (r *errorReader) Read(p []byte) (int, error) {
	return 0, errRead
}

func TestNewRequestModifier(t *testing.T) {
	h := http.Header{}
	h.Set("foo", "bar")
	req := &http.Request{Header: h}
	modifier := NewRequestModifier(req)
	st.Expect(t, modifier.Request, req)
	st.Expect(t, modifier.Header, h)
}

func TestReadString(t *testing.T) {
	bodyStr := `{"hello":"bonjour"}`
	strReader := strings.NewReader(bodyStr)
	body := ioutil.NopCloser(strReader)
	req := &http.Request{Header: http.Header{}, Body: body}
	modifier := NewRequestModifier(req)
	str, err := modifier.ReadString()
	st.Expect(t, err, nil)
	st.Expect(t, str, bodyStr)

	body = ioutil.NopCloser(&errorReader{})
	req = &http.Request{Header: http.Header{}, Body: body}
	modifier = NewRequestModifier(req)
	str, err = modifier.ReadString()
	st.Expect(t, err, errRead)
	st.Expect(t, str, "")
}

func TestReadBytes(t *testing.T) {
	bodyBytes := []byte(`{"hello":"bonjour"}`)
	strReader := bytes.NewBuffer(bodyBytes)
	body := ioutil.NopCloser(strReader)
	req := &http.Request{Header: http.Header{}, Body: body}
	modifier := NewRequestModifier(req)
	str, err := modifier.ReadBytes()
	st.Expect(t, err, nil)
	st.Expect(t, str, bodyBytes)

	body = ioutil.NopCloser(&errorReader{})
	req = &http.Request{Header: http.Header{}, Body: body}
	modifier = NewRequestModifier(req)
	buf, err := modifier.ReadBytes()
	st.Expect(t, err, errRead)
	st.Expect(t, len(buf), 0)
}
