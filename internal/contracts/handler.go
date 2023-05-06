package contracts

import (
	"net/url"

	"github.com/evg4b/uncors/internal/urlreplacer"
)

type URLReplacerFactory interface {
	Make(requestURL *url.URL) (*urlreplacer.Replacer, *urlreplacer.Replacer, error)
}
