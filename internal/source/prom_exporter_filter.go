package source

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/davidjspooner/dsflow/pkg/job"
	"github.com/davidjspooner/net-mapper/internal/framework"
)

type httpFilter struct {
	url     *url.URL
	metrics []string
	client  *http.Client
}

var _ Filter = (*httpFilter)(nil)

func init() {
	Register("prom_exporter_filter", newHttpFilter)
}

func newHttpFilter(args framework.Config) (Source, error) {
	h := &httpFilter{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}

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
	return "prom_exporter_filter"
}

func (h *httpFilter) responceContainsMetric(r io.ReadCloser, u *url.URL) bool {
	defer r.Close()
	//read the lines and look for the metrics
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		first_word := strings.FieldsFunc(line, func(r rune) bool { return r == ' ' || r == '{' })[0]
		if slices.Contains(h.metrics, first_word) {
			log.Printf("Found metric %q in %s\n", first_word, u.String())
			return true
		}
		_ = line
	}
	return false
}

func (h *httpFilter) testHost(ctx context.Context, u *url.URL) error {
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return err
	}
	req = req.WithContext(ctx)
	resp, err := h.client.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode/100 != 2 {
		resp.Body.Close()
		return err
	}
	if h.responceContainsMetric(resp.Body, u) {
		return nil
	}
	return fmt.Errorf("no matching metrics found")
}

func (h *httpFilter) Filter(ctx context.Context, input HostList) (HostList, error) {
	output := make(HostList, 0, len(input))
	lock := sync.Mutex{}

	executer := job.NewExecuter[string](log.Default())

	executer.Start(ctx, 10, func(ctx context.Context, host string) error {
		u := *h.url
		port := u.Port()
		if port == "" {
			u.Host = host
		} else {
			u.Host = host + ":" + port
		}
		err := h.testHost(ctx, &u)
		if err != nil {
			return nil
		}
		lock.Lock()
		defer lock.Unlock()
		output = append(output, host)
		return nil
	}, input)

	err := executer.WaitForCompletion()
	if err != nil {
		return nil, err
	}

	return output, nil
}
