package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type Response struct {
	Status  int
	Headers http.Header
	Body    []byte
	Delay   time.Duration
}

func (r *Response) UnmarshalJSON(data []byte) error {
	type alias Response
	v := struct {
		alias
		Body    string
		Headers map[string]string
	}{}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	r.Status = v.Status
	r.Delay = v.Delay
	r.Body = []byte(v.Body)
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
		Headers map[string]string
	}{
		Headers: make(map[string]string),
	}

	v.Delay = time.Duration(r.Delay) / time.Millisecond
	v.Status = r.Status
	v.Body = string(r.Body)
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

		kv := strings.Split(line, ":")
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
		Body:    []byte(body),
	}, nil
}

func (r *Response) Write(w http.ResponseWriter) error {
	for key := range r.Headers {
		w.Header().Set(key, r.Headers.Get(key))
	}
	w.WriteHeader(r.Status)
	_, err := w.Write(r.Body)

	return err
}

type Responses map[string]*Response

func (rs Responses) String(key string) string {
	r := rs.Get(key)
	if r == nil {
		return ""
	}

	return fmt.Sprintf("%s > %d", key, r.Status)
}

func (rs Responses) FromString(s string) *Response {
	split := strings.Split(s, ">")
	if len(split) < 2 {
		return nil
	}

	key := strings.Trim(split[0], " ")
	return rs.Get(key)
}

func (rs Responses) Get(key string) *Response {
	return rs[key]
}

func (rs Responses) SaveResponsesToPath(path string) error {
	f, err := openConfigFile(path)
	if err != nil {
		return err
	}
	defer f.Close()

	return rs.SaveResponsesToFile(f)
}

func (rs Responses) SaveResponsesToFile(f *os.File) error {
	buf, err := json.MarshalIndent(struct {
		Responses Responses
	}{rs}, "", "  ")
	if err != nil {
		return err
	}

	if _, err := f.Write(buf); err != nil {
		return err
	}

	return nil
}

func LoadResponsesFromPath(path string) (Responses, error) {
	f, err := openConfigFile(path)
	if err != nil {
		return nil, err
	}

	return LoadResponsesFromFile(f)
}

func LoadResponsesFromFile(f *os.File) (Responses, error) {
	r := struct {
		Responses Responses
	}{}

	if err := json.NewDecoder(f).Decode(&r); err != nil {
		if err == io.EOF {
			return nil, nil
		}
		return nil, err
	}

	return r.Responses, nil
}

func openConfigFile(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0666)
}
