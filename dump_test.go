package httplab

import (
	"bytes"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"strings"
	//"sort"
	"sort"
)

func TestDumpRequestWithJSON(t *testing.T) {
	t.Run("should be indented", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/withJSON", bytes.NewBuffer(
			[]byte(`{"foo": "bar", "a": [1,2,3]}`),
		))
		req.Header.Set("Content-Type", "application/json")

		buf, err := DumpRequest(req)
		require.NoError(t, err)
		fmt.Printf("%s\n", buf)
	})

	t.Run("should be displayed as is", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/invalidJSON", bytes.NewBuffer(
			[]byte(`invalid json`),
		))
		req.Header.Set("Content-Type", "application/json")

		buf, err := DumpRequest(req)
		require.NoError(t, err)
		fmt.Printf("%s\n", buf)
	})
}

func TestDumpRequestHeaders(t *testing.T) {
	t.Run("request headers should be dumped in sorted order", func(t *testing.T) {

		keys := []string{"B", "A", "C", "D", "E", "F", "H", "G", "I"}
		req, _ := http.NewRequest("GET", "/", bytes.NewBuffer(nil))
		for _, k := range keys {
			req.Header.Set(k, "")
		}

		buf, err := DumpRequest(req)
		require.NoError(t, err)
		sort.Strings(keys)

		startLine := "GET / HTTP/1.1\n"
		response :=  startLine + strings.Join(keys, ": \n") + ": \n"

		assert.Contains(t, response, string(Decolorize(buf)))
	})
}

func TestDecolorization(t *testing.T) {
	for i, _ := range [107]struct{}{} {
		text := "Some Text"
		nocolor := Decolorize([]byte(withColor(i, text)))
		assert.Equal(t, text, string(nocolor))
	}
}
