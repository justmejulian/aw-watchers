// Package aw is a minimal client for the ActivityWatch REST API.
package aw

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	base string
	http *http.Client
}

// Event is an ActivityWatch event. Posting an event with an ID replaces it.
type Event struct {
	ID        *int64         `json:"id,omitempty"`
	Timestamp time.Time      `json:"timestamp"`
	Duration  float64        `json:"duration"` // seconds
	Data      map[string]any `json:"data"`
}

// New returns a client for the local server; testing uses aw-qt's --testing port.
func New(testing bool) *Client {
	port := 5600
	if testing {
		port = 5666
	}
	return &Client{
		base: fmt.Sprintf("http://127.0.0.1:%d/api/0", port),
		http: &http.Client{Timeout: 5 * time.Second},
	}
}

func (c *Client) do(method, path string, body, out any) error {
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, c.base+path, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	// 304: bucket already exists
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotModified {
		msg, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("%s %s: %s: %s", method, path, resp.Status, msg)
	}
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}

// Hostname returns the hostname the server reports, used in bucket names.
func (c *Client) Hostname() (string, error) {
	var info struct {
		Hostname string `json:"hostname"`
	}
	if err := c.do(http.MethodGet, "/info", nil, &info); err != nil {
		return "", err
	}
	return info.Hostname, nil
}

// CreateBucket creates the bucket if it does not exist yet.
func (c *Client) CreateBucket(id, eventType, client, hostname string) error {
	body := map[string]string{"client": client, "type": eventType, "hostname": hostname}
	return c.do(http.MethodPost, "/buckets/"+id, body, nil)
}

// InsertEvent inserts the event (or replaces it when it has an ID) and
// returns it with the ID assigned by the server.
func (c *Client) InsertEvent(bucket string, e Event) (Event, error) {
	e.Timestamp = e.Timestamp.UTC().Truncate(time.Microsecond)
	var inserted []Event
	if err := c.do(http.MethodPost, "/buckets/"+bucket+"/events", []Event{e}, &inserted); err != nil {
		return e, err
	}
	if len(inserted) == 0 {
		return e, fmt.Errorf("insert into %s: empty response", bucket)
	}
	return inserted[0], nil
}
