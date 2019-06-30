// +build !go1.8

package lambdarouter

import "net/url"

func unescape(path string) (string, error) {
	return url.QueryUnescape(path)
}
