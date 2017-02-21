// +build visualtest

package main

import (
	"bytes"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDumpRequestWithJSON(t *testing.T) {
	body := bytes.NewBuffer(nil)
	req, _ := http.NewRequest("GET", "/foo", body)
	req.Header.Set("X-Server", "HTTPLab")
	req.Header.Set("Content-Type", "application/json")

	t.Run("Valid", func(t *testing.T) {
		body.WriteString(`{"foo": "bar", "a": [1,2,3]}`)
		buf, err := DumpRequest(req)
		fmt.Println(string(buf))
		assert.NoError(t, err)
	})

	t.Run("Invalid", func(t *testing.T) {
		body.Reset()
		body.WriteString(`some invalid json`)
		buf, err := DumpRequest(req)
		fmt.Println(string(buf))
		assert.NoError(t, err)
	})

}
