package httplab

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
	Location: http://foo.bar:8000
	X-Empty: 
	Invalid
	`

	resp, err := NewResponse("", headers, "")
	require.NoError(t, err)
	assert.Equal(t, "application/json", resp.Headers.Get("Content-Type"))
	assert.Equal(t, "value", resp.Headers.Get("X-MyHeader"))
	assert.Equal(t, "", resp.Headers.Get("X-Empty"))
	assert.Equal(t, "http://foo.bar:8000", resp.Headers.Get("Location"))
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
		Body: Body{
			Input: []byte("Hello, World"),
		},
	}

	resp.Write(rec)

	assert.Equal(t, resp.Status, rec.Code)
	assert.Equal(t, resp.Headers.Get("X-Foo"), rec.Header().Get("X-Foo"))
	assert.Equal(t, resp.Body.Payload(), rec.Body.Bytes())
}

func TestResponsesList(t *testing.T) {
	rl := NewResponsesList()
	rl.Add("200", &Response{Status: 200}).
		Add("201", &Response{Status: 201}).
		Add("404", &Response{Status: 404}).
		Add("500", &Response{Status: 500})

	t.Run("Len()", func(t *testing.T) {
		assert.Equal(t, 4, rl.Len())
	})

	t.Run("Get()", func(t *testing.T) {
		assert.Equal(t, 200, rl.Get("200").Status)
		assert.Equal(t, 201, rl.Get("201").Status)
		assert.Equal(t, 404, rl.Get("404").Status)
		assert.Equal(t, 500, rl.Get("500").Status)
	})

	t.Run("Indexing", func(t *testing.T) {
		assert.Equal(t, 200, rl.Cur().Status)
		assert.Equal(t, 0, rl.Index())

		rl.Next()
		assert.Equal(t, 201, rl.Cur().Status)
		assert.Equal(t, 1, rl.Index())

		rl.Next()
		assert.Equal(t, 404, rl.Cur().Status)
		assert.Equal(t, 2, rl.Index())

		rl.Next()
		assert.Equal(t, 500, rl.Cur().Status)
		assert.Equal(t, 3, rl.Index())

		rl.Next()
		assert.Equal(t, 200, rl.Cur().Status)
		assert.Equal(t, 0, rl.Index())

		rl.Prev()
		assert.Equal(t, 500, rl.Cur().Status)
		assert.Equal(t, 3, rl.Index())

		rl.Prev()
		assert.Equal(t, 404, rl.Cur().Status)
		assert.Equal(t, 2, rl.Index())
	})

	t.Run("Del()", func(t *testing.T) {
		for _, status := range []string{"200", "201", "404", "500"} {
			assert.NotNil(t, rl.Get(status))
			rl.Del(status)
			assert.Nil(t, rl.Get(status))
		}
	})
}

func TestLoadFromJSON(t *testing.T) {
	rl := NewResponsesList()
	require.NoError(t, rl.Load("./testdata/httplab.json"))

	r := rl.Get("t1")
	require.NotNil(t, r)
	assert.Equal(t, 200, r.Status)
	assert.Equal(t, time.Duration(1000), r.Delay)
	assert.Equal(t, "value", r.Headers.Get("X-MyHeader"))

	r.Body.Mode = BodyInput
	assert.Equal(t, []byte("xxx"), r.Body.Payload())
	assert.Equal(t, []byte("size: 3 bytes\n"), r.Body.Info())

	r.Body.Mode = BodyFile
	assert.Equal(t, []byte("<html></html>"), r.Body.Payload())

	t.Run("When config file is empty", func(t *testing.T) {
		path := string(time.Now().UnixNano())
		defer os.Remove(path)

		require.NoError(t, rl.Load(path))
		assert.Equal(t, 0, rl.Len())

		// file has to be created
		_, err := os.Stat(path)
		assert.NoError(t, err)
	})
}

func TestExpandPathExpansion(t *testing.T) {
	defer os.Setenv("HOME", os.Getenv("HOME"))

	for key, val := range map[string]string{
		"HOME": "/home/gchaincl",
		"ENV1": "env1",
		"ENV2": "env2",
	} {
		os.Setenv(key, val)
	}

	paths := []struct {
		expr     string
		expected string
	}{
		{"~/foo", "/home/gchaincl/foo"},
		{"./foo/~/bar", "./foo/~/bar"},
		{"/$ENV1/foo/$ENV2", "/env1/foo/env2"},
		{"$NOTDEFINED/foo", "/foo"},
	}

	for _, path := range paths {
		assert.Equal(t, path.expected, ExpandPath(path.expr))
	}
}
