package main

import (
	"context"
	"net/http"
	"strconv"
)

type Enclosure struct {
	URL    string `xml:"url,attr,omitempty"`
	Type   string `xml:"type,attr,omitempty"`
	Length int64  `xml:"length,attr,omitempty"`
}

func NewEnclosure(
	ctx context.Context,
	enclosureURL string,
) (*Enclosure, error) {
	e := &Enclosure{URL: enclosureURL}

	req, err := http.NewRequestWithContext(ctx, "HEAD", e.URL, nil)
	if err != nil {
		return e, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return e, err
	}

	if v := resp.Header.Get("Content-Type"); v != "" {
		e.Type = v
	}

	if v := resp.Header.Get("Content-Length"); v != "" {
		length, err := strconv.ParseInt(v, 10, 64)
		if err == nil {
			e.Length = length
		}
	}

	return e, nil
}
