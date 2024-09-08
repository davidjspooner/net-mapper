package publisher

import (
	"context"
	"fmt"

	"github.com/davidjspooner/net-mapper/internal/framework"

	"time"
)

type filePublisher struct {
	fileName string
}

var _ Interface = &filePublisher{}

func init() {
	Register("file", newFilePublisher)
}

func newFilePublisher(args framework.Config) (Interface, error) {
	p := &filePublisher{}
	err := framework.CheckFields(args, "filename")
	if err != nil {
		return nil, err
	}

	p.fileName, err = framework.ConsumeArg[string](args, "filename")
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (p *filePublisher) Publish(ctx context.Context, report string, generated time.Time) error {
	return fmt.Errorf("file publisher not implemented")
}
