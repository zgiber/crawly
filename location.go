package crawly

const (
	internalLink = "internal"
	staticLink   = "static"
	externalLink = "external"
)

// required: page's url, title, static assets, internal links and external links.
type location struct {
	URL      string
	Title    string
	Links    []*htmlLink
	Children []*location
	Err      error
}

type htmlLink struct {
	Typ string
	Src string
}

// TODO: don't forget tests for this guy
func (loc *location) childByURL(u string) *location {
	for _, childLocation := range loc.Children {
		if childLocation.URL == u {
			return childLocation
		}
		return childLocation.childByURL(u)
	}
	return nil
}
