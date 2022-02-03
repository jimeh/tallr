package main

import (
	"context"
	"sort"
	"sync"
	"time"

	"go.uber.org/multierr"
)

type Fetcher struct {
	mux         sync.Mutex
	URL         string
	PageLimit   int
	Concurrency int

	Items       []*Item
	ItemURLs    map[string]struct{}
	CachedUntil time.Time
}

func (f *Fetcher) Run(ctx context.Context) error {
	f.mux.Lock()
	defer f.mux.Unlock()

	if f.Concurrency == 0 {
		f.Concurrency = 1
	}
	if f.PageLimit == 0 {
		f.PageLimit = 1
	}

	var wg sync.WaitGroup
	var rwg sync.WaitGroup

	var errs error
	jobChan := make(chan string)
	resultChan := make(chan *Item)
	errChan := make(chan error)

	rwg.Add(1)
	go func() {
		defer rwg.Done()
		for err := range errChan {
			errs = multierr.Append(errs, err)
		}
	}()

	rwg.Add(1)
	go func() {
		defer rwg.Done()
		for item := range resultChan {
			f.Items = append(f.Items, item)
		}
	}()

	for i := 0; i < f.Concurrency; i++ {
		wg.Add(1)
		go f.worker(ctx, &wg, jobChan, resultChan, errChan)
	}

	nextURL := f.URL
	n := 1
	for {
		page, err := NewListPage(ctx, nextURL)
		if err != nil {
			errChan <- err
			continue
		}

		newItems := 0
		for _, i := range page.Items {
			if i.PublishDate.After(f.CachedUntil) {
				newItems++
				jobChan <- i.URL
			}
		}

		nextURL = page.NextPageURL

		n++
		if nextURL == "" || n > f.PageLimit || newItems == 0 {
			break
		}
	}

	close(jobChan)
	wg.Wait()

	close(resultChan)
	close(errChan)
	rwg.Wait()

	if len(f.Items) > 0 {
		sort.Slice(f.Items, func(i, j int) bool {
			return f.Items[i].PublishDate.After(f.Items[j].PublishDate)
		})
		f.CachedUntil = f.Items[0].PublishDate
	}

	return errs
}

func (f *Fetcher) worker(
	ctx context.Context,
	wg *sync.WaitGroup,
	jobs chan string,
	results chan *Item,
	errs chan error,
) {
	defer wg.Done()
	for itemURL := range jobs {
		item, err := NewItem(ctx, itemURL)
		if err != nil {
			errs <- err
			continue
		}

		results <- item
	}
}
