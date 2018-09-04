package crawly

import (
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"testing"
)

func TestParseHTML(t *testing.T) {
	f, err := os.Open("example.html")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	b, err := ioutil.ReadAll(f)
	if err != nil {
		t.Fatal(err)
	}

	lastURL, _ := url.Parse("https://for_internal_links")
	title, links, err := parseHTML(lastURL, b)
	fmt.Println("title", title)
	for _, l := range links {
		fmt.Println(l.Typ, l.Src)
	}

	// TODO: auto tests: create the test document with known cases
	// and add assertions
}
