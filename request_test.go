package intercept

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"github.com/nbio/st"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
)

var errRead = errors.New("read error")

type errorReader struct{}

func (r *errorReader) Read(p []byte) (int, error) {
	return 0, errRead
}

type user struct {
	XMLName xml.Name `xml:"Person"`
	Name    string
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
	bodyStr := `{"name":"Rick"}`
	strReader := strings.NewReader(bodyStr)
	body := ioutil.NopCloser(strReader)
	req := &http.Request{Header: http.Header{}, Body: body}
	modifier := NewRequestModifier(req)
	str, err := modifier.ReadString()
	st.Expect(t, err, nil)
	st.Expect(t, str, bodyStr)
}

func TestReadStringError(t *testing.T) {
	body := ioutil.NopCloser(&errorReader{})
	req := &http.Request{Header: http.Header{}, Body: body}
	modifier := NewRequestModifier(req)
	str, err := modifier.ReadString()
	st.Expect(t, err, errRead)
	st.Expect(t, str, "")
}

func TestReadBytes(t *testing.T) {
	bodyBytes := []byte(`{"name":"Rick"}`)
	strReader := bytes.NewBuffer(bodyBytes)
	body := ioutil.NopCloser(strReader)
	req := &http.Request{Header: http.Header{}, Body: body}
	modifier := NewRequestModifier(req)
	str, err := modifier.ReadBytes()
	st.Expect(t, err, nil)
	st.Expect(t, str, bodyBytes)
}

func TestReadBytesError(t *testing.T) {
	body := ioutil.NopCloser(&errorReader{})
	req := &http.Request{Header: http.Header{}, Body: body}
	modifier := NewRequestModifier(req)
	buf, err := modifier.ReadBytes()
	st.Expect(t, err, errRead)
	st.Expect(t, len(buf), 0)
}

func TestDecodeJSON(t *testing.T) {
	bodyBytes := []byte(`{"name":"Rick"}`)
	strReader := bytes.NewBuffer(bodyBytes)
	body := ioutil.NopCloser(strReader)
	req := &http.Request{Header: http.Header{}, Body: body}
	modifier := NewRequestModifier(req)
	u := user{}
	err := modifier.DecodeJSON(&u)
	st.Expect(t, err, nil)
	st.Expect(t, u.Name, "Rick")
}

func TestDecodeJSONErrorFromReadBytes(t *testing.T) {
	body := ioutil.NopCloser(&errorReader{})
	req := &http.Request{Header: http.Header{}, Body: body}
	modifier := NewRequestModifier(req)
	u := user{}
	err := modifier.DecodeJSON(&u)
	st.Expect(t, err, errRead)
	st.Expect(t, u.Name, "")
}

func TestDecodeJSONEOF(t *testing.T) {
	bodyBytes := []byte("")
	strReader := bytes.NewBuffer(bodyBytes)
	body := ioutil.NopCloser(strReader)
	req := &http.Request{Header: http.Header{}, Body: body}
	modifier := NewRequestModifier(req)
	u := user{}
	err := modifier.DecodeJSON(&u)
	st.Expect(t, err, nil)
	st.Expect(t, u.Name, "")
}

func TestDecodeJSONErrorFromDecode(t *testing.T) {
	bodyBytes := []byte(`/`)
	strReader := bytes.NewBuffer(bodyBytes)
	body := ioutil.NopCloser(strReader)
	req := &http.Request{Header: http.Header{}, Body: body}
	modifier := NewRequestModifier(req)
	u := user{}
	err := modifier.DecodeJSON(&u)
	_, ok := (err).(*json.SyntaxError)
	st.Expect(t, ok, true)
	st.Expect(t, err.Error(), "invalid character '/' looking for beginning of value")
	st.Expect(t, u.Name, "")
}

func TestDecodeXML(t *testing.T) {
	bodyBytes := []byte(`<Person><Name>Rick</Name></Person>`)
	strReader := bytes.NewBuffer(bodyBytes)
	body := ioutil.NopCloser(strReader)
	req := &http.Request{Header: http.Header{}, Body: body}
	modifier := NewRequestModifier(req)
	u := user{}
	err := modifier.DecodeXML(&u, nil)
	st.Expect(t, err, nil)
	st.Expect(t, u.Name, "Rick")
}

func TestDecodeXMLErrorFromReadBytes(t *testing.T) {
	body := ioutil.NopCloser(&errorReader{})
	req := &http.Request{Header: http.Header{}, Body: body}
	modifier := NewRequestModifier(req)
	u := user{}
	err := modifier.DecodeXML(&u, nil)
	st.Expect(t, err, errRead)
	st.Expect(t, u.Name, "")
}

func TestDecodeXMLErrorFromDecode(t *testing.T) {
	bodyBytes := []byte(`]]>`)
	strReader := bytes.NewBuffer(bodyBytes)
	body := ioutil.NopCloser(strReader)
	req := &http.Request{Header: http.Header{}, Body: body}
	modifier := NewRequestModifier(req)
	u := user{}
	err := modifier.DecodeXML(&u, nil)
	_, ok := (err).(*xml.SyntaxError)
	st.Expect(t, ok, true)
	st.Expect(t, err.Error(), "XML syntax error on line 1: unescaped ]]> not in CDATA section")
	st.Expect(t, u.Name, "")
}

func TestDecodeXMLEOF(t *testing.T) {
	bodyBytes := []byte("")
	strReader := bytes.NewBuffer(bodyBytes)
	body := ioutil.NopCloser(strReader)
	req := &http.Request{Header: http.Header{}, Body: body}
	modifier := NewRequestModifier(req)
	u := user{}
	err := modifier.DecodeXML(&u, nil)
	st.Expect(t, err, nil)
	st.Expect(t, u.Name, "")
}

func TestBytes(t *testing.T) {
	req := &http.Request{Header: http.Header{}}
	modifier := NewRequestModifier(req)
	modifier.Bytes([]byte("hello"))
	modifiedBody, err := ioutil.ReadAll(req.Body)
	st.Expect(t, err, nil)
	st.Expect(t, modifiedBody, []byte("hello"))
}

func TestStringGet(t *testing.T) {
	req := &http.Request{Method: "GET", Header: http.Header{}}
	modifier := NewRequestModifier(req)
	modifier.String("hello")
	st.Expect(t, req.Body, nil)
}

func TestStringHead(t *testing.T) {
	req := &http.Request{Method: "HEAD", Header: http.Header{}}
	modifier := NewRequestModifier(req)
	modifier.String("hello")
	st.Expect(t, req.Body, nil)
}

func TestString(t *testing.T) {
	req := &http.Request{Method: "POST", Header: http.Header{}}
	modifier := NewRequestModifier(req)
	modifier.String("hello")
	modifiedBody, err := ioutil.ReadAll(req.Body)
	st.Expect(t, err, nil)
	st.Expect(t, modifiedBody, []byte("hello"))
}
