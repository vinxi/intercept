package intercept

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
)

// XMLCharDecoder is a helper type that takes a stream of bytes (not encoded in
// UTF-8) and returns a reader that encodes the bytes into UTF-8. This is done
// because Go's XML library only supports XML encoded in UTF-8.
type XMLCharDecoder func(charset string, input io.Reader) (io.Reader, error)

// ReqModifierFunc represent the function interface for request modifiers.
type ReqModifierFunc func(*RequestModifier)

// Filter defines whether a RequestModifier should be applied or not.
type Filter func(*http.Request) bool

// RequestModifier implements a convenient abstraction to modify an http.Request,
// including methods to read, decode/encode and define JSON/XML/String/Binary bodies
// and modify HTTP headers.
type RequestModifier struct {
	// Header exposes the request http.Header type.
	Header http.Header

	// Request exposes the current http.Request to be modified.
	Request *http.Request
}

// NewRequestModifier creates a new request modifier that modifies the given http.Request.
func NewRequestModifier(req *http.Request) *RequestModifier {
	return &RequestModifier{Request: req, Header: req.Header}
}

// ReadString reads the whole body of the current http.Request and returns it as string.
func (s *RequestModifier) ReadString() (string, error) {
	buf, err := ioutil.ReadAll(s.Request.Body)
	if err != nil {
		return "", err
	}
	s.Bytes(buf)
	return string(buf), nil
}

// ReadBytes reads the whole body of the current http.Request and returns it as bytes.
func (s *RequestModifier) ReadBytes() ([]byte, error) {
	buf, err := ioutil.ReadAll(s.Request.Body)
	if err != nil {
		return nil, err
	}
	s.Bytes(buf)
	return buf, nil
}

// DecodeJSON reads and parses the current http.Request body and tries to decode it as JSON.
func (s *RequestModifier) DecodeJSON(userStruct interface{}) error {
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

// DecodeXML reads and parses the current http.Request body and tries to decode it as XML.
func (s *RequestModifier) DecodeXML(userStruct interface{}, charsetReader XMLCharDecoder) error {
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

// Bytes sets the given bytes as http.Request body.
func (s *RequestModifier) Bytes(body []byte) {
	s.Request.Body = ioutil.NopCloser(bytes.NewReader(body))
}

// String sets the given string as http.Request body.
func (s *RequestModifier) String(body string) {
	if s.Request.Method == "GET" || s.Request.Method == "HEAD" {
		return
	}
	s.Request.Body = ioutil.NopCloser(bytes.NewReader([]byte(body)))
}

// JSON sets the given JSON serializable struct as http.Request body
// defining the proper content length header.
func (s *RequestModifier) JSON(data interface{}) error {
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

	s.Request.Body = ioutil.NopCloser(buf)
	s.Request.ContentLength = int64(buf.Len())
	s.Request.Header.Set("Content-Type", "application/json")
	return nil
}

// XML sets the given XML serializable struct as http.Request body
// defining the proper content length header.
func (s *RequestModifier) XML(data interface{}) error {
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

	s.Request.Body = ioutil.NopCloser(buf)
	s.Request.ContentLength = int64(buf.Len())
	s.Request.Header.Set("Content-Type", "application/xml")
	return nil
}

// Reader sets the given io.Reader stream as http.Request body
// defining the proper content length header.
func (s *RequestModifier) Reader(body io.Reader) error {
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

// RequestInterceptor interceps a given http.Request using a custom request modifier function.
type RequestInterceptor struct {
	Modifier ReqModifierFunc
	Filters  []Filter
}

// Request intercepts an HTTP request and passes it to the given request modifier function.
func Request(h ReqModifierFunc) *RequestInterceptor {
	return &RequestInterceptor{Modifier: h, Filters: []Filter{}}
}

// Filter intercepts an HTTP requests if and only if the given filter returns true.
func (s *RequestInterceptor) Filter(f ...Filter) {
	s.Filters = append(s.Filters, f...)
}

// HandleHTTP handles the middleware call chain, intercepting the request data if possible.
// This methods implements the middleware layer compatible interface.
func (s *RequestInterceptor) HandleHTTP(w http.ResponseWriter, r *http.Request, h http.Handler) {
	if s.filter(r) {
		req := NewRequestModifier(r)
		s.Modifier(req)
	}
	h.ServeHTTP(w, r)
}

func (s RequestInterceptor) filter(req *http.Request) bool {
	for _, filter := range s.Filters {
		if !filter(req) {
			return false
		}
	}
	return true
}
