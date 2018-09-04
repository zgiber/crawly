package crawly

import (
	"encoding/xml"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
)

// Node represents an arbitrary xml element in the document
type Node struct {
	XMLName xml.Name
	Attrs   []xml.Attr `xml:"-"`
	Content []byte     `xml:",innerxml"`
	Nodes   []*Node    `xml:",any"`
}

// UnmarshalXML Implements xml.Unmarshaler
func (n *Node) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	n.Attrs = start.Attr
	type node Node

	return d.DecodeElement((*node)(n), &start)
}

func walk(nodes []*Node, f func(*Node)) {
	for _, n := range nodes {
		f(n)
		walk(n.Nodes, f)
	}
}

func parseHTML(lastURL *url.URL, body []byte) (title string, links []*htmlLink, err error) {
	n := &Node{}
	err = xml.Unmarshal(body, n)
	if err != nil {
		return
	}

	var titleIsSet bool
	walk([]*Node{n}, func(n *Node) {
		if !titleIsSet {
			if title, err = extractTitle(n); err == nil {
				titleIsSet = true
			}
		}

		if link, extractErr := extractLink(lastURL, n); extractErr == nil {
			links = append(links, link)
		}
	})
	return
}

func normaliseLink(lastURL *url.URL, link *htmlLink) error {
	if strings.HasPrefix(link.Src, "//") {
		// protocol relative link
		link.Src = strings.Join([]string{lastURL.Scheme, link.Src}, ":")

	}

	if strings.HasPrefix(link.Src, "/") {
		// relative link
		link.Src = fmt.Sprintf("%s://%s%s", lastURL.Scheme, lastURL.Host, link.Src)
	}

	//
	// set link type to static|internal|external ...
	// really a best effort approach - TODO: fix it
	//

	knownStaticExtensions := []string{".jpg", ".jpeg", ".png", ".ico", ".mp3", ".ogg", ".svg", ".gif", ".mpg", ".mp4", ".mpeg", ".avi", ".mp4"} // TODO: add more
	ext := filepath.Ext(link.Src)
	if contains(knownStaticExtensions, ext) {
		link.Typ = staticLink
		return nil
	}

	internal, err := isInternal(lastURL, link.Src)
	if err != nil {
		return err
	}

	if internal {
		link.Typ = internalLink
	} else {
		link.Typ = externalLink
	}

	//
	// TODO: spend some time and collect edge cases
	//

	return nil
}

func extractLink(lastURL *url.URL, n *Node) (*htmlLink, error) {

	link := &htmlLink{}
	isAnchor := n.XMLName.Local == "a"
	val, hasAttr := attr(n, "href")
	if isAnchor && hasAttr {
		// <a> nodes with href attr which does not end in some known extension
		// or <link> nodes
		link.Src = val
	} else if val, hasAttr := attr(n, "src"); hasAttr {
		// the rest
		link.Src = val
	} else {
		return nil, errors.New("not a link")
	}

	//
	// TODO: unhandled cases ?
	//

	err := normaliseLink(lastURL, link)
	return link, err
}

func attr(n *Node, attr string) (string, bool) {
	for _, a := range n.Attrs {
		if a.Name.Local == attr {
			return a.Value, true
		}
	}
	return "", false
}

func contains(ss []string, s string) bool {
	for _, item := range ss {
		if item == s {
			return true
		}
	}
	return false
}

func isInternal(lastURL *url.URL, linkURL string) (bool, error) {
	linkParsedURL, err := url.Parse(linkURL)
	if err != nil {
		return false, err
	}

	return lastURL.Host == linkParsedURL.Host, nil
}

func extractTitle(n *Node) (string, error) {
	if strings.ToLower(n.XMLName.Local) == "title" {
		return string(n.Content), nil
	}

	return "", errors.New("not a title")
}
