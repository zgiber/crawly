// +build integration

package crawly

import (
	"encoding/json"
	"os"
	"testing"
	"time"
)

// It is not nice to use Wikipedia resources
// TODO: host a site with proper content for testing
func TestCrawl(t *testing.T) {
	url := "https://en.wikipedia.org/wiki/Main_Page"
	req := &Request{
		URL:      url,
		Timeout:  5 * time.Second,
		MaxPages: 100,
	}
	resp, err := Crawl(req)
	if err != nil {
		t.Fatal(err)
	}

	b, _ := json.MarshalIndent(resp.Root, "", "  ")
	_ = b

	outFile, err := os.Create("out.json")
	outFile.Write(b)
}
