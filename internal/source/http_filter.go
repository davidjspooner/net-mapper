package source

import (
	"context"
	"fmt"
	"net/url"

	"github.com/davidjspooner/net-mapper/internal/framework"
)

type httpFilter struct {
	url     *url.URL
	metrics []string
}

var _ Filter = (*httpFilter)(nil)

func init() {
	Register("http", newHttpFilter)
}

func newHttpFilter(args framework.Config) (Source, error) {
	h := &httpFilter{}

	err := framework.CheckFields(args, "url", "metric")
	if err != nil {
		return nil, err
	}
	urlString, err := framework.ConsumeArg[string](args, "url")
	if err != nil {
		return nil, err
	}
	if urlString == "" {
		return nil, fmt.Errorf("url is empty")
	}
	h.url, err = url.Parse(urlString)
	if err != nil {
		return nil, err
	}
	if h.url.Scheme == "" {
		return nil, fmt.Errorf("url %s has no scheme", urlString)
	}
	if h.url.Path == "" {
		return nil, fmt.Errorf("url %s has no path", urlString)
	}

	h.metrics, err = framework.ConsumeOptionalArg[[]string](args, "metric", []string{})
	if err != nil {
		return nil, err
	}
	for _, m := range h.metrics {
		err = framework.IsIdentifier(m)
		if err != nil {
			return nil, fmt.Errorf("metric %q is invalid: %s", m, err)
		}
	}

	return h, nil
}

func (h *httpFilter) Kind() string {
	return "http"
}

func (h *httpFilter) Filter(ctx context.Context, input HostList) (HostList, error) {
	return nil, fmt.Errorf("http condition not implemented")
}
