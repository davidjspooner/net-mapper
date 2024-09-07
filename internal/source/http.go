package source

import (
	"context"
	"fmt"
	"net/url"

	"github.com/davidjspooner/net-mapper/internal/framework"
)

type Http struct {
	url    *url.URL
	metric string
}

var _ Filter = (*Http)(nil)

func init() {
	Register("http", newHttpFilter)
}

func newHttpFilter(args framework.Config) (Source, error) {
	h := &Http{}

	err := framework.CheckKeys(args, "url", "metric")
	if err != nil {
		return nil, err
	}
	urlString, err := framework.GetArg(args, "url", "")
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

	h.metric, err = framework.GetArg(args, "metric", "")
	if err != nil {
		return nil, err
	}
	err = framework.IsIdentifier(h.metric)
	if err != nil {
		return nil, fmt.Errorf("metric name %s is invalid: %s", h.metric, err)
	}

	return h, nil
}

func (h *Http) Kind() string {
	return "http"
}

func (h *Http) Filter(ctx context.Context, input HostList) (HostList, error) {
	return nil, fmt.Errorf("http condition not implemented")
}
