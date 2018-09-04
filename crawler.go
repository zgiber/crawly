package crawly

import (
	"context"
	"io/ioutil"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// Request ...
type Request struct {
	URL      string
	Timeout  time.Duration
	Delay    time.Duration
	MaxPages int
}

// Response ...
type Response struct {
	Root *location
}

type crawler struct {
	rootURL           *url.URL
	maxPages          int
	timeout           time.Duration
	delay             time.Duration
	requestDoneSignal chan struct{}
	rateLimit         chan struct{}
	stop              func()
}

// Crawl the site and return a *Response.
// error is returned if req.URL is invalid
func Crawl(req *Request) (*Response, error) {
	ctx, cancel := context.WithTimeout(context.Background(), req.Timeout)
	defer cancel()

	u, err := url.Parse(req.URL)
	if err != nil {
		return nil, err
	}

	cr := &crawler{
		rootURL:           u,
		timeout:           req.Timeout,
		delay:             req.Delay,
		requestDoneSignal: make(chan struct{}, req.MaxPages),
		// rateLimit:         make(chan struct{}, 1),
		stop: cancel,
	}

	response := &Response{}
	response.Root = cr.crawl(ctx, req.URL) // crawl the site

	return response, nil
}

// TODO:
// func controlRate(ctx context.Context, delay time.Duration, signal chan struct{}) {
// 	if delay == 0 {
// 		delay = 1000 * time.Millisecond
// 		// TBD: this is definitely too fast,
// 		// there should be a more sensible default delay to avoid DDoS-ing the servers
// 	}

// 	t := time.NewTicker(delay)
// 	defer t.Stop()

// 	for {
// 		select {
// 		case <-ctx.Done():
// 			return
// 		default:
// 		}

// 		select {
// 		case <-t.C:
// 			signal <- struct{}{}
// 		}
// 	}
// }

func (cr *crawler) crawl(ctx context.Context, rawURL string) *location {
	select {
	case <-ctx.Done():
		// context is already canceled, don't even start
		return nil
	default:
		// continue
	}

	// retrieve root page
	// (err can be ignored as the possible error conditions
	// are already checked for earlier)
	wg := sync.WaitGroup{}
	root := cr.fetchLocation(ctx, &wg, rawURL)
	wg.Wait()
	cr.stop()
	return root
}

func (cr *crawler) fetchLocation(ctx context.Context, wg *sync.WaitGroup, rawURL string) *location {

	loc := &location{URL: rawURL}

	// don't fetch more pages after stopped
	select {
	case <-ctx.Done():
		return nil
	default:
	}

	select {
	case cr.requestDoneSignal <- struct{}{}:
	default:
		// max pages have been reached, don't fetch this
		return nil
	}

	wg.Add(1)
	go func() {
		defer wg.Done()

		// Mutating location parameters after it's pointer returned
		// could be a race but this func is the only one doing it
		// so it's safe here

		req, err := cr.makeHTTPRequest(ctx, rawURL)
		if err != nil {
			loc.Err = NewError("failed to create HTTP request")
			return
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			loc.Err = NewError("failed to retrieve page")
			return
		}

		if resp.StatusCode/100 != 2 {
			loc.Err = NewError("unexpected status")
			return
		}

		// ensure that client can be reused by subsequent calls
		defer resp.Body.Close()

		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			loc.Err = NewError("failed to read response body")
			return
		}

		// keep the last used url and use it for normalising relative links
		// this is valid URL unless the request would have failed
		lastURL, _ := url.Parse(resp.Request.URL.String())
		loc.Title, loc.Links, loc.Err = parseHTML(lastURL, b)
		if loc.Err != nil {
			loc.Err = NewError("failed to parse response body")
			return
		}

		// fetch child pages for internal links
		for _, link := range loc.Links {
			if link.Typ == internalLink {
				childLoc := cr.fetchLocation(ctx, wg, link.Src)
				if childLoc != nil {
					loc.Children = append(loc.Children, childLoc)
				}
			}
		}
	}()

	return loc
}

func (cr *crawler) makeHTTPRequest(ctx context.Context, url string) (*http.Request, error) {
	// prepare http request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	return req.WithContext(ctx), nil
}
