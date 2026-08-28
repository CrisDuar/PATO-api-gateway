package proxy

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

func NewReverseProxy(target string) (*httputil.ReverseProxy, error) {

	targetURL, err := url.Parse(target)
	if err != nil {
		return nil, err
	}

	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	originalDirector := proxy.Director

	proxy.Director = func(req *http.Request) {

		originalDirector(req)

		req.URL.Path = "/api/v1" + strings.TrimPrefix(
			req.URL.Path,
			"/api",
		)

		req.Host = targetURL.Host
	}

	return proxy, nil
}
