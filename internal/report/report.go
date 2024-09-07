package report

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"text/template"

	"github.com/davidjspooner/net-mapper/internal/framework"
	"github.com/davidjspooner/net-mapper/internal/source"
)

type templatedReport struct {
	extra        framework.Config
	textTemplate *template.Template
}

func init() {
	Register("template", newTemplatedReport)
	Register("", newTemplatedReport)
}

func newTemplatedReport(args framework.Config) (Interface, error) {
	//user can add arbitrary data to the report

	r := &templatedReport{}

	filename, err := framework.GetArg(args, "template_file", "")
	if err != nil {
		return nil, err
	}

	templateText, err := framework.GetArg(args, "template_inline", "")
	if err != nil {
		return nil, err
	}
	if filename != "" && templateText != "" {
		return nil, fmt.Errorf("template_file and template_inline are mutually exclusive")
	}
	if filename == "" && templateText == "" {
		return nil, fmt.Errorf("either template_file or template_inline is required")
	}

	if filename != "" {
		content, err := os.ReadFile(filename)
		if err != nil {
			return nil, err
		}
		templateText = string(content)
	}

	r.textTemplate, err = template.New("template").Parse(templateText)
	if err != nil {
		return nil, err
	}

	delete(args, "template_file")
	delete(args, "template_inline")
	r.extra = args

	return r, nil
}

func (r *templatedReport) Generate(ctx context.Context, hosts source.HostList) (string, error) {
	var err error
	buffer := bytes.Buffer{}

	data := framework.Config{
		"hosts": hosts,
		"extra": r.extra,
	}

	err = r.textTemplate.Execute(&buffer, &data)
	if err != nil {
		return "", err
	}
	return buffer.String(), nil
}
