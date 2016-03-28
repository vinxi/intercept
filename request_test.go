package intercept

import (
	"github.com/nbio/st"
	"net/http"
	"testing"
)

func TestNewRequestModifier(t *testing.T) {
	h := http.Header{}
	h.Set("foo", "bar")
	req := &http.Request{Header: h}
	modifier := NewRequestModifier(req)
	st.Expect(t, modifier.Request, req)
	st.Expect(t, modifier.Header, h)
}
