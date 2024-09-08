package publisher

import (
	"context"
	"fmt"
	"time"

	"github.com/davidjspooner/net-mapper/internal/framework"
)

type kubernetesPublisher struct {
	fileName string
}

var _ Interface = &kubernetesPublisher{}

func init() {
	Register("kubernetes", newKubernetesPublisher)
}

func newKubernetesPublisher(args framework.Config) (Interface, error) {
	p := &kubernetesPublisher{}

	err := framework.CheckFields(args, "filename")
	if err != nil {
		return nil, err
	}
	p.fileName, err = framework.ConsumeOptionalArg(args, "filename", "")
	if err != nil {
		return nil, err
	}
	if p.fileName == "" {
		return nil, fmt.Errorf("filename is required")
	}

	return p, nil
}

func (p *kubernetesPublisher) Publish(ctx context.Context, report string, generated time.Time) error {
	return fmt.Errorf("kubernetes publisher not implemented")
}
