package intercept

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"github.com/nbio/st"
	"gopkg.in/vinci-proxy/utils.v0"
	"io"
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
	XMLName xml.Name `xml:"Person" json:"-"`
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
	req := &http.Request{Body: body}
	modifier := NewRequestModifier(req)
	str, err := modifier.ReadString()
	st.Expect(t, err, nil)
	st.Expect(t, str, bodyStr)
}

func TestReadStringError(t *testing.T) {
	body := ioutil.NopCloser(&errorReader{})
	req := &http.Request{Body: body}
	modifier := NewRequestModifier(req)
	str, err := modifier.ReadString()
	st.Expect(t, err, errRead)
	st.Expect(t, str, "")
}

func TestReadBytes(t *testing.T) {
	bodyBytes := []byte(`{"name":"Rick"}`)
	strReader := bytes.NewBuffer(bodyBytes)
	body := ioutil.NopCloser(strReader)
	req := &http.Request{Body: body}
	modifier := NewRequestModifier(req)
	str, err := modifier.ReadBytes()
	st.Expect(t, err, nil)
	st.Expect(t, str, bodyBytes)
}

func TestReadBytesError(t *testing.T) {
	body := ioutil.NopCloser(&errorReader{})
	req := &http.Request{Body: body}
	modifier := NewRequestModifier(req)
	buf, err := modifier.ReadBytes()
	st.Expect(t, err, errRead)
	st.Expect(t, len(buf), 0)
}

func TestDecodeJSON(t *testing.T) {
	bodyBytes := []byte(`{"name":"Rick"}`)
	strReader := bytes.NewBuffer(bodyBytes)
	body := ioutil.NopCloser(strReader)
	req := &http.Request{Body: body}
	modifier := NewRequestModifier(req)
	u := user{}
	err := modifier.DecodeJSON(&u)
	st.Expect(t, err, nil)
	st.Expect(t, u.Name, "Rick")
}

func TestDecodeJSONErrorFromReadBytes(t *testing.T) {
	body := ioutil.NopCloser(&errorReader{})
	req := &http.Request{Body: body}
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
	req := &http.Request{Body: body}
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
	req := &http.Request{Body: body}
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
	req := &http.Request{Body: body}
	modifier := NewRequestModifier(req)
	u := user{}
	err := modifier.DecodeXML(&u, nil)
	st.Expect(t, err, nil)
	st.Expect(t, u.Name, "Rick")
}

func TestDecodeXMLErrorFromReadBytes(t *testing.T) {
	body := ioutil.NopCloser(&errorReader{})
	req := &http.Request{Body: body}
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
	req := &http.Request{Body: body}
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
	req := &http.Request{Body: body}
	modifier := NewRequestModifier(req)
	u := user{}
	err := modifier.DecodeXML(&u, nil)
	st.Expect(t, err, nil)
	st.Expect(t, u.Name, "")
}

func TestBytes(t *testing.T) {
	req := &http.Request{}
	modifier := NewRequestModifier(req)
	modifier.Bytes([]byte("hello"))
	modifiedBody, err := ioutil.ReadAll(req.Body)
	st.Expect(t, err, nil)
	st.Expect(t, string(modifiedBody), "hello")
}

func TestStringGet(t *testing.T) {
	req := &http.Request{Method: "GET"}
	modifier := NewRequestModifier(req)
	modifier.String("hello")
	st.Expect(t, req.Body, nil)
}

func TestStringHead(t *testing.T) {
	req := &http.Request{Method: "HEAD"}
	modifier := NewRequestModifier(req)
	modifier.String("hello")
	st.Expect(t, req.Body, nil)
}

func TestString(t *testing.T) {
	req := &http.Request{Method: "POST"}
	modifier := NewRequestModifier(req)
	modifier.String("hello")
	modifiedBody, err := ioutil.ReadAll(req.Body)
	st.Expect(t, err, nil)
	st.Expect(t, string(modifiedBody), "hello")
}

func TestJSONWithStructAsParameter(t *testing.T) {
	req := &http.Request{Header: http.Header{}}
	modifier := NewRequestModifier(req)
	u := &user{Name: "Rick"}
	err := modifier.JSON(u)
	st.Expect(t, err, nil)
	modifiedBody, err := ioutil.ReadAll(req.Body)
	st.Expect(t, err, nil)
	expectedBody := "{\"Name\":\"Rick\"}\n"
	st.Expect(t, string(modifiedBody), expectedBody)
	st.Expect(t, req.ContentLength, int64(len(expectedBody)))
	st.Expect(t, req.Header.Get("Content-Type"), "application/json")
}

func TestJSONWithStringAsParameter(t *testing.T) {
	req := &http.Request{Header: http.Header{}}
	modifier := NewRequestModifier(req)
	input := `{"Name":"Rick"}`
	err := modifier.JSON(input)
	st.Expect(t, err, nil)
	modifiedBody, err := ioutil.ReadAll(req.Body)
	st.Expect(t, err, nil)
	st.Expect(t, string(modifiedBody), input)
	st.Expect(t, req.ContentLength, int64(len(input)))
	st.Expect(t, req.Header.Get("Content-Type"), "application/json")
}

func TestJSONWithBytesAsParameter(t *testing.T) {
	req := &http.Request{Header: http.Header{}}
	modifier := NewRequestModifier(req)
	input := []byte(`{"Name":"Rick"}`)
	err := modifier.JSON(input)
	st.Expect(t, err, nil)
	modifiedBody, err := ioutil.ReadAll(req.Body)
	st.Expect(t, err, nil)
	st.Expect(t, modifiedBody, input)
	st.Expect(t, req.ContentLength, int64(len(input)))
	st.Expect(t, req.Header.Get("Content-Type"), "application/json")
}

func TestJSONEncodingError(t *testing.T) {
	req := &http.Request{Header: http.Header{}}
	modifier := NewRequestModifier(req)
	input := make(map[int]int)
	err := modifier.JSON(input)
	_, ok := err.(*json.UnsupportedTypeError)
	st.Expect(t, ok, true)
	st.Expect(t, err.Error(), "json: unsupported type: map[int]int")
}

func TestXMLWithStructAsParameter(t *testing.T) {
	req := &http.Request{Header: http.Header{}}
	modifier := NewRequestModifier(req)
	u := &user{Name: "Rick"}
	err := modifier.XML(u)
	st.Expect(t, err, nil)
	modifiedBody, err := ioutil.ReadAll(req.Body)
	st.Expect(t, err, nil)
	expectedBody := `<Person><Name>Rick</Name></Person>`
	st.Expect(t, string(modifiedBody), expectedBody)
	st.Expect(t, req.ContentLength, int64(len(expectedBody)))
	st.Expect(t, req.Header.Get("Content-Type"), "application/xml")
}

func TestXMLWithStringAsParameter(t *testing.T) {
	req := &http.Request{Header: http.Header{}}
	modifier := NewRequestModifier(req)
	input := `<Person><Name>Rick</Name></Person>`
	err := modifier.XML(input)
	st.Expect(t, err, nil)
	modifiedBody, err := ioutil.ReadAll(req.Body)
	st.Expect(t, err, nil)
	st.Expect(t, string(modifiedBody), input)
	st.Expect(t, req.ContentLength, int64(len(input)))
	st.Expect(t, req.Header.Get("Content-Type"), "application/xml")
}

func TestXMLWithBytesAsParameter(t *testing.T) {
	req := &http.Request{Header: http.Header{}}
	modifier := NewRequestModifier(req)
	input := []byte(`<Person><Name>Rick</Name></Person>`)
	err := modifier.XML(input)
	st.Expect(t, err, nil)
	modifiedBody, err := ioutil.ReadAll(req.Body)
	st.Expect(t, err, nil)
	st.Expect(t, string(modifiedBody), string(input))
	st.Expect(t, req.ContentLength, int64(len(input)))
	st.Expect(t, req.Header.Get("Content-Type"), "application/xml")
}

func TestXMLEncodingError(t *testing.T) {
	req := &http.Request{Header: http.Header{}}
	modifier := NewRequestModifier(req)
	input := make(map[int]int)
	err := modifier.XML(input)
	_, ok := err.(*xml.UnsupportedTypeError)
	st.Expect(t, ok, true)
	st.Expect(t, err.Error(), "xml: unsupported type: map[int]int")
}

func TestReaderWithBufferAsParameter(t *testing.T) {
	req := &http.Request{}
	modifier := NewRequestModifier(req)
	reader := bytes.NewBuffer([]byte("Hello"))
	err := modifier.Reader(reader)
	st.Expect(t, err, nil)
	_, ok := req.Body.(io.ReadCloser)
	st.Expect(t, ok, true)
	body, _ := ioutil.ReadAll(req.Body)
	st.Expect(t, string(body), "Hello")
}

func TestReaderWithReaderAsParameter(t *testing.T) {
	req := &http.Request{}
	modifier := NewRequestModifier(req)
	reader := bytes.NewReader([]byte("Hello"))
	err := modifier.Reader(reader)
	st.Expect(t, err, nil)
	_, ok := req.Body.(io.ReadCloser)
	st.Expect(t, ok, true)
	body, _ := ioutil.ReadAll(req.Body)
	st.Expect(t, string(body), "Hello")
}

func TestReaderWithStringReaderAsParameter(t *testing.T) {
	req := &http.Request{}
	modifier := NewRequestModifier(req)
	reader := strings.NewReader("Hello")
	err := modifier.Reader(reader)
	st.Expect(t, err, nil)
	_, ok := req.Body.(io.ReadCloser)
	st.Expect(t, ok, true)
	body, _ := ioutil.ReadAll(req.Body)
	st.Expect(t, string(body), "Hello")
}

func TestRequest(t *testing.T) {
	intercepted := false
	modifierFunc := func(m *RequestModifier) {
		intercepted = true
	}
	interceptor := Request(modifierFunc)
	interceptor.Modifier(&RequestModifier{})
	st.Expect(t, intercepted, true)
}

func TestHandleHTTP(t *testing.T) {
	interceptor := Request(func(m *RequestModifier) {
		m.Header.Set("foo", "bar")
		m.String("Hello")
	})
	stubbedWriter := utils.NewWriterStub()
	req := &http.Request{Method: "POST", Header: make(http.Header)}
	handler := http.HandlerFunc(func(writer http.ResponseWriter, r *http.Request) {
		st.Expect(t, writer, stubbedWriter)
		st.Expect(t, r.Header.Get("foo"), "bar")
		requestBody, _ := ioutil.ReadAll(r.Body)
		st.Expect(t, string(requestBody), "Hello")
	})
	interceptor.HandleHTTP(stubbedWriter, req, handler)
}
