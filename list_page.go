package main

import (
	"context"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type ListItem struct {
	URL         string
	PublishDate time.Time
}

type ListPage struct {
	URL         string
	PrevPageURL string
	NextPageURL string
	Items       []*ListItem

	parsed bool
}

func NewListPage(
	ctx context.Context,
	listURL string,
) (*ListPage, error) {
	ip := &ListPage{URL: listURL}

	err := ip.Parse(ctx)
	if err != nil {
		return nil, err
	}

	return ip, nil
}

func (lp *ListPage) Parse(ctx context.Context) error {
	if lp.parsed {
		return ErrAlreadyParsed
	}

	req, err := http.NewRequestWithContext(ctx, "GET", lp.URL, nil)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return err
	}

	lp.parseListItems(doc)
	lp.parseNextPageURL(doc)
	lp.parsePrevPageURL(doc)

	lp.parsed = true

	return nil
}

func (lp *ListPage) parseListItems(doc *goquery.Document) {
	doc.Find("section").Each(func(_ int, sec *goquery.Selection) {
		href, ok := sec.Find("a[data-link-name=article]").Attr("href")
		if !ok || href == "" {
			return
		}

		li := &ListItem{URL: href}

		sec.Find("time").EachWithBreak(func(_ int, t *goquery.Selection) bool {
			timestamp, ok := t.Attr("data-timestamp")
			if !ok {
				return true
			}

			millisec, err := strconv.ParseInt(timestamp, 10, 64)
			if err != nil {
				return true
			}
			li.PublishDate = time.Unix(millisec/1000, millisec%1000)

			return false
		})

		lp.Items = append(lp.Items, li)
	})

	sort.Slice(lp.Items, func(i, j int) bool {
		return lp.Items[i].PublishDate.After(lp.Items[j].PublishDate)
	})
}

func (lp *ListPage) parseNextPageURL(doc *goquery.Document) {
	if href, ok := doc.Find(".pagination a[rel=next]").Attr("href"); ok {
		lp.NextPageURL = href
	}
}

func (lp *ListPage) parsePrevPageURL(doc *goquery.Document) {
	if href, ok := doc.Find(".pagination a[rel=prev]").Attr("href"); ok {
		lp.PrevPageURL = href
	}
}
