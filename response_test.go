package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResponseStatus(t *testing.T) {
	// only status between 100 and 599 are valid
	for i := 100; i < 600; i++ {
		status := strconv.Itoa(i)
		_, err := NewResponse(status, "", "")
		assert.NoError(t, err)
	}

	for _, status := range []string{"600", "99", "foo", "2xx"} {
		_, err := NewResponse(status, "", "")
		assert.Error(t, err, fmt.Sprintf("status '%s' should be invalid", status))
	}

	for _, format := range []string{" %d ", "%d\n", " %d \n", "%d\r\r"} {
		status := fmt.Sprintf(format, 200)
		_, err := NewResponse(status, "", "")
		assert.NoError(t, err)
	}

	// default value
	t.Run("Default Value", func(t *testing.T) {
		resp, err := NewResponse("", "", "")
		require.NoError(t, err)
		assert.Equal(t, 200, resp.Status)
	})
}

func TestResponseHeaders(t *testing.T) {
	headers := `
	Content-Type: application/json
	X-MyHeader: value
	X-Empty: 
	Invalid
	`

	resp, err := NewResponse("", headers, "")
	require.NoError(t, err)
	assert.Equal(t, "application/json", resp.Headers.Get("Content-Type"))
	assert.Equal(t, "value", resp.Headers.Get("X-MyHeader"))
	assert.Equal(t, "", resp.Headers.Get("X-Empty"))
	assert.Contains(t, resp.Headers, "X-Empty")
	assert.NotContains(t, resp.Headers, "Invalid")
}

func TestResponseWrite(t *testing.T) {
	rec := httptest.NewRecorder()
	resp := &Response{
		Status: 201,
		Headers: http.Header{
			"X-Foo": []string{"bar"},
		},
		Body: []byte("Hello, World"),
	}

	resp.Write(rec)

	assert.Equal(t, resp.Status, rec.Code)
	assert.Equal(t, resp.Headers.Get("X-Foo"), rec.Header().Get("X-Foo"))
	assert.Equal(t, resp.Body, rec.Body.Bytes())
}

func TestLoadFromJSON(t *testing.T) {
	rs, err := LoadResponsesFromPath("./testdata/httplab.json")
	require.NoError(t, err)

	require.Contains(t, rs, "t1")

	r := rs["t1"]
	assert.Equal(t, 200, r.Status)
	assert.Equal(t, time.Duration(1000), r.Delay)
	assert.Equal(t, []byte("xxx"), r.Body)
	assert.Equal(t, "value", r.Headers.Get("X-MyHeader"))

	t.Run("When file is empty", func(t *testing.T) {
		path := time.Now().Format(time.UnixDate)
		defer os.Remove(path)

		rs, err := LoadResponsesFromPath(path)
		require.NoError(t, err)
		assert.Len(t, rs, 0)

		// file has to be created
		_, err = os.Stat(path)
		assert.NoError(t, err)
	})
}
