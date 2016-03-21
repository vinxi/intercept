package intercept

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

// ResModifierFunc defines the function interface for http.Response modifiers.
type ResModifierFunc func(*ResponseModifier)

// ResponseModifier implements a convenient abstraction to modify an http.Response,
// including methods to read, decode/encode and define JSON/XML/String/Binary bodies
// and modify HTTP headers.
type ResponseModifier struct {
	Header   http.Header
	Request  *http.Request
	Response *http.Response
}

// NewResponseModifier creates a new response modifier that modifies the given http.Response.
func NewResponseModifier(req *http.Request, res *http.Response) *ResponseModifier {
	return &ResponseModifier{Request: req, Response: res, Header: res.Header}
}

// Status sets a new status code in the http.Response to be modified.
func (s *ResponseModifier) Status(status int) {
	s.Response.StatusCode = status
	s.Response.Status = strconv.Itoa(status) + " " + http.StatusText(status)
}

// ReadString reads the whole body of the current http.Response and returns it as string.
func (s *ResponseModifier) ReadString() (string, error) {
	buf, err := ioutil.ReadAll(s.Response.Body)
	if err != nil {
		return "", err
	}
	s.Bytes(buf)
	return string(buf), nil
}

// ReadBytes reads the whole body of the current http.Response and returns it as bytes.
func (s *ResponseModifier) ReadBytes() ([]byte, error) {
	buf, err := ioutil.ReadAll(s.Response.Body)
	if err != nil {
		return nil, err
	}
	s.Bytes(buf)
	return buf, nil
}

// DecodeJSON reads and parses the current http.Response body and tries to decode it as JSON.
func (s *ResponseModifier) DecodeJSON(userStruct interface{}) error {
	buf, err := s.ReadBytes()
	if err != nil {
		return err
	}

	jsonDecoder := json.NewDecoder(bytes.NewReader(buf))
	err = jsonDecoder.Decode(&userStruct)
	if err != nil && err != io.EOF {
		return err
	}

	return nil
}

// DecodeXML reads and parses the current http.Response body and tries to decode it as XML.
func (s *ResponseModifier) DecodeXML(userStruct interface{}, charsetReader XMLCharDecoder) error {
	buf, err := s.ReadBytes()
	if err != nil {
		return err
	}

	xmlDecoder := xml.NewDecoder(bytes.NewReader(buf))
	if charsetReader != nil {
		xmlDecoder.CharsetReader = charsetReader
	}
	if err := xmlDecoder.Decode(&userStruct); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// String sets the given string as http.Response body.
func (s *ResponseModifier) String(body string) {
	s.Response.Body = ioutil.NopCloser(bytes.NewReader([]byte(body)))
}

// Bytes sets the given bytes as http.Response body.
func (s *ResponseModifier) Bytes(body []byte) {
	s.Response.Body = ioutil.NopCloser(bytes.NewReader(body))
}

// JSON sets the given JSON serializable struct as http.Response body
// defining the proper content length header.
func (s *ResponseModifier) JSON(data interface{}) error {
	buf := &bytes.Buffer{}

	switch data.(type) {
	case string:
		buf.WriteString(data.(string))
	case []byte:
		buf.Write(data.([]byte))
	default:
		if err := json.NewEncoder(buf).Encode(data); err != nil {
			return err
		}
	}

	s.Response.Body = ioutil.NopCloser(buf)
	s.Response.ContentLength = int64(buf.Len())
	s.Response.Header.Set("Content-Type", "application/json")
	return nil
}

// XML sets the given XML serializable struct as http.Response body
// defining the proper content length header.
func (s *ResponseModifier) XML(data interface{}) error {
	buf := &bytes.Buffer{}

	switch data.(type) {
	case string:
		buf.WriteString(data.(string))
	case []byte:
		buf.Write(data.([]byte))
	default:
		if err := xml.NewEncoder(buf).Encode(data); err != nil {
			return err
		}
	}

	s.Response.Body = ioutil.NopCloser(buf)
	s.Response.ContentLength = int64(buf.Len())
	s.Response.Header.Set("Content-Type", "application/xml")
	return nil
}

// Reader sets the given io.Reader stream as http.Response body
// defining the proper content length header.
func (s *ResponseModifier) Reader(body io.Reader) error {
	rc, ok := body.(io.ReadCloser)
	if !ok && body != nil {
		rc = ioutil.NopCloser(body)
	}

	req := s.Request
	if body != nil {
		switch v := body.(type) {
		case *bytes.Buffer:
			req.ContentLength = int64(v.Len())
		case *bytes.Reader:
			req.ContentLength = int64(v.Len())
		case *strings.Reader:
			req.ContentLength = int64(v.Len())
		}
	}

	req.Body = rc
	return nil
}

// WriterInterceptor implements an http.ResponseWriter compatible interface that will intercept and buffer
// any method call until the body writes is completed, and then will call the http.Response modifier
// function to intercept and modify it accordingly before writting the final response fields.
type WriterInterceptor struct {
	closed        bool
	headerWritten bool
	buf           []byte
	mutex         *sync.Mutex
	response      *http.Response
	modifier      ResModifierFunc
	writer        http.ResponseWriter
}

// NewWriterInterceptor creates a new http.ResponseWriter capable interface
// that will intercept the current response.
func NewWriterInterceptor(w http.ResponseWriter, req *http.Request, fn ResModifierFunc) *WriterInterceptor {
	res := &http.Response{
		Request:    req,
		StatusCode: 200,
		Status:     "200 OK",
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     make(http.Header),
		Body:       ioutil.NopCloser(bytes.NewReader([]byte{})),
	}
	return &WriterInterceptor{mutex: &sync.Mutex{}, writer: w, modifier: fn, response: res}
}

// Header returns the current response http.Header.
func (w *WriterInterceptor) Header() http.Header {
	return w.response.Header
}

// WriteHeader intercepts the desired response status code.
func (w *WriterInterceptor) WriteHeader(status int) {
	w.response.StatusCode = status
	w.response.Status = strconv.Itoa(status) + " " + http.StatusText(status)
}

// Write intercepts and stores chunks of bytes as part of the response body.
func (w *WriterInterceptor) Write(b []byte) (int, error) {
	length := w.response.Header.Get("Content-Length")
	if length == "" || length == "0" {
		w.buf = b
		return w.DoWrite()
	}

	w.response.ContentLength += int64(len(b))
	w.buf = append(w.buf, b...)

	// If not EOF
	if cl, _ := strconv.Atoi(length); w.response.ContentLength != int64(cl) {
		return len(b), nil
	}

	w.response.Body = ioutil.NopCloser(bytes.NewReader(w.buf))
	resm := NewResponseModifier(w.response.Request, w.response)
	w.modifier(resm)
	return w.DoWrite()
}

// Close closes the body readers and flags the interceptor as closed status.
func (w *WriterInterceptor) Close() {
	w.mutex.Lock()
	if w.closed {
		return
	}
	w.closed = true
	w.buf = nil
	w.response.Body.Close()
	w.mutex.Unlock()
}

// DoWrite writes the final HTTP response header and body in the real http.ResponseWriter.
func (w *WriterInterceptor) DoWrite() (int, error) {
	w.writeHeader()
	return w.writeBody()
}

// writeHeader writes the final response header fields.
func (w *WriterInterceptor) writeHeader() {
	if w.headerWritten || w.closed {
		return
	}

	if w.response.StatusCode != 0 {
		w.writer.WriteHeader(w.response.StatusCode)
	}

	target := w.writer.Header()
	for k, v := range w.response.Header {
		target[k] = v
	}

	w.headerWritten = true
}

// writeBody writes the final response body.
func (w *WriterInterceptor) writeBody() (int, error) {
	if w.closed {
		return 0, nil
	}

	buf, err := ioutil.ReadAll(w.response.Body)
	defer w.Close()
	if err != nil {
		return 0, err
	}
	return w.writer.Write(buf)
}

// Response intercepts an HTTP response and passes it to the given response modifier function.
func Response(fn ResModifierFunc) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "OPTIONS" || r.Method == "HEAD" {
				h.ServeHTTP(w, r)
				return
			}

			writer := NewWriterInterceptor(w, r, fn)
			defer h.ServeHTTP(writer, r)

			notifier, ok := w.(http.CloseNotifier)
			if !ok {
				return
			}

			notify := notifier.CloseNotify()
			go func() {
				<-notify
				writer.Close()
			}()
		})
	}
}
