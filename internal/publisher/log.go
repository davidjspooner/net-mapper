package publisher

import (
	"context"
	"log"

	"github.com/davidjspooner/net-mapper/internal/framework"

	"time"
)

type logPublisher struct {
	reportName     string
	prefix, suffix string
}

var _ Interface = &logPublisher{}

func init() {
	Register("log", newLogPublisher)
}

func newLogPublisher(args framework.Config) (Interface, error) {
	p := &logPublisher{}

	err := framework.CheckFields(args, "report", "prefix", "suffix")
	if err != nil {
		return nil, err
	}

	p.prefix, err = framework.ConsumeOptionalArg(args, "prefix", "")
	if err != nil {
		return nil, err
	}
	p.suffix, err = framework.ConsumeOptionalArg(args, "suffix", "")
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (p *logPublisher) Publish(ctx context.Context, report string, generated time.Time) error {
	log.Printf("%s%s%s", p.prefix, report, p.suffix)
	return nil
}

func (p *logPublisher) ReportName() string {
	return p.reportName
}
