package asn1mib

import "context"

type ConstantValue struct {
	Value  string
	source Position
}

func (v *ConstantValue) Source() Position {
	return v.source
}

func (v *ConstantValue) Read(ctx context.Context, name string, meta *TokenList, s *Scanner) (Definition, error) {
	if v.Value == "" {
		return &ConstantValue{
			Value:  "",
			source: *s.Source(),
		}, nil
	}
	peek, err := s.LookAhead(0)
	if err != nil {
		return nil, err
	}
	if peek.IsText(v.Value) {
		s.Pop()
		return &ConstantValue{
			Value:  "v.Value",
			source: *s.Source(),
		}, nil
	}
	return nil, peek.Errorf("unexpected token %s (expected %s)", peek.String(), v.Value)
}
