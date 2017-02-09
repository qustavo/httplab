package httplab

import (
	"bytes"
	"fmt"
	"net/http"
	"testing"
)

func TestDumpRequest(t *testing.T) {
	body := bytes.NewBufferString(`
	{"foo": "bar", "a": [1,2,3]}
	`)

	req, _ := http.NewRequest("GET", "/foo", body)
	req.Header.Set("X-Server", "HTTPLab")
	req.Header.Set("Content-Type", "application/json")
	fmt.Println(string(DumpRequest(req)))
}
