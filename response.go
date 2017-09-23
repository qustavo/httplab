package httplab

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

// BodyMode represent the current Body mode
type BodyMode uint

func (m BodyMode) String() string {
	switch m {
	case BodyInput:
		return "Input"
	case BodyFile:
		return "File"
	}
	return ""
}

const (
	BodyInput BodyMode = iota + 1
	BodyFile
)

type Body struct {
	Mode  BodyMode
	Input []byte
	File  *os.File
}

func (body *Body) Payload() []byte {
	switch body.Mode {
	case BodyInput:
		return body.Input
	case BodyFile:
		if body.File == nil {
			return nil
		}

		// XXX: Handle this error
		bytes, _ := ioutil.ReadAll(body.File)
		body.File.Seek(0, 0)
		return bytes
	}
	return nil
}

func (body *Body) Info() []byte {
	switch body.Mode {
	case BodyInput:
		return body.Input
	case BodyFile:
		if body.File == nil {
			return nil
		}

		// XXX: Handle this error
		stats, _ := body.File.Stat()
		w := &bytes.Buffer{}
		fmt.Fprintf(w, "file: %s\n", body.File.Name())
		fmt.Fprintf(w, "size: %d bytes\n", stats.Size())
		fmt.Fprintf(w, "perm: %s\n", stats.Mode())
		return w.Bytes()
	}
	return nil
}

func (body *Body) SetFile(path string) error {
	file, err := os.Open(ExpandPath(path))
	if err != nil {
		return err
	}

	body.File = file
	body.Mode = BodyFile
	return nil
}

type Response struct {
	Status  int
	Headers http.Header
	Body    Body
	Delay   time.Duration
}

func (r *Response) UnmarshalJSON(data []byte) error {
	type alias Response
	v := struct {
		alias
		Body    string
		File    string
		Headers map[string]string
	}{}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	r.Status = v.Status
	r.Delay = v.Delay
	r.Body.Input = []byte(v.Body)
	if v.File != "" {
		if err := r.Body.SetFile(v.File); err != nil {
			return err
		}
	}

	if r.Body.File != nil {
		r.Body.Mode = BodyFile
	} else {
		r.Body.Mode = BodyInput
	}

	if r.Headers == nil {
		r.Headers = http.Header{}
	}
	for key := range v.Headers {
		r.Headers.Set(key, v.Headers[key])
	}

	return nil
}

func (r *Response) MarshalJSON() ([]byte, error) {
	type alias Response
	v := struct {
		alias
		Body    string
		File    string
		Headers map[string]string
	}{
		Headers: make(map[string]string),
	}

	v.Delay = time.Duration(r.Delay) / time.Millisecond
	v.Status = r.Status

	if len(r.Body.Input) > 0 {
		v.Body = string(r.Body.Input)
	}

	if r.Body.File != nil {
		v.File = r.Body.File.Name()
	}

	for key := range r.Headers {
		v.Headers[key] = r.Headers.Get(key)
	}

	return json.MarshalIndent(v, "", "  ")
}

func NewResponse(status, headers, body string) (*Response, error) {
	// Parse Status
	status = strings.Trim(status, " \r\n")
	if status == "" {
		status = "200"
	}
	code, err := strconv.Atoi(status)
	if err != nil {
		return nil, fmt.Errorf("Status: %v", err)
	}

	if code < 100 || code > 599 {
		return nil, fmt.Errorf("Status should be between 100 and 599")
	}

	// Parse Headers
	hdr := http.Header{}
	lines := strings.Split(headers, "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		kv := strings.SplitN(line, ":", 2)
		if len(kv) != 2 {
			continue
		}
		key := strings.TrimSpace(kv[0])
		val := strings.TrimSpace(kv[1])
		hdr.Set(key, val)
	}

	return &Response{
		Status:  code,
		Headers: hdr,
		Body: Body{
			Input: []byte(body),
		},
	}, nil
}

func (r *Response) Write(w http.ResponseWriter) error {
	for key := range r.Headers {
		w.Header().Set(key, r.Headers.Get(key))
	}
	w.WriteHeader(r.Status)
	_, err := w.Write(r.Body.Payload())

	return err
}

type ResponsesList struct {
	List    map[string]*Response
	keys    []string
	current int
}

func NewResponsesList() *ResponsesList {
	return (&ResponsesList{}).reset()
}

func (rl *ResponsesList) reset() *ResponsesList {
	rl.current = 0
	rl.List = make(map[string]*Response)
	rl.keys = nil
	return rl
}

func (rl *ResponsesList) load(path string) (map[string]*Response, error) {
	f, err := openConfigFile(path)
	if err != nil {
		return nil, err
	}

	rs := struct {
		Responses map[string]*Response
	}{}

	if err := json.NewDecoder(f).Decode(&rs); err != nil {
		if err == io.EOF {
			return nil, nil
		}
		return nil, err
	}

	return rs.Responses, nil
}

func (rl *ResponsesList) Load(path string) error {
	rs, err := rl.load(path)
	if err != nil {
		return err
	}

	rl.reset()
	if rs != nil {
		rl.List = rs
	}

	for key, _ := range rs {
		rl.keys = append(rl.keys, key)
	}
	sort.Strings(rl.keys)

	return nil
}

func (rl *ResponsesList) Save(path string) error {
	f, err := openConfigFile(path)
	if err != nil {
		return err
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return err
	}

	if err := f.Truncate(stat.Size()); err != nil {
		return err
	}

	buf, err := json.MarshalIndent(struct {
		Responses map[string]*Response
	}{rl.List}, "", "  ")
	if err != nil {
		return err
	}

	if _, err := f.Write(buf); err != nil {
		return err
	}

	return nil
}

func (rl *ResponsesList) Next()                    { rl.current = (rl.current + 1) % len(rl.keys) }
func (rl *ResponsesList) Prev()                    { rl.current = (rl.current - 1 + len(rl.keys)) % len(rl.keys) }
func (rl *ResponsesList) Cur() *Response           { return rl.List[rl.keys[rl.current]] }
func (rl *ResponsesList) Index() int               { return rl.current }
func (rl *ResponsesList) Len() int                 { return len(rl.keys) }
func (rl *ResponsesList) Keys() []string           { return rl.keys }
func (rl *ResponsesList) Get(key string) *Response { return rl.List[key] }
func (rl *ResponsesList) Add(key string, r *Response) *ResponsesList {
	rl.keys = append(rl.keys, key)
	sort.Strings(rl.keys)
	rl.List[key] = r
	return rl
}

func (rl *ResponsesList) Del(key string) bool {
	if _, ok := rl.List[key]; !ok {
		return false
	}
	delete(rl.List, key)

	i := sort.SearchStrings(rl.keys, key)
	rl.keys = append(rl.keys[:i], rl.keys[i+1:]...)

	return true
}

func ExpandPath(path string) string {
	if path[0] == '~' {
		path = "$HOME" + path[1:len(path)]
	}
	return os.ExpandEnv(path)
}

func openConfigFile(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0666)
}
