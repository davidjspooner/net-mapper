package mibdb

import (
	"context"
	"fmt"

	"github.com/davidjspooner/net-mapper/pkg/asn1/asn1error"
	"github.com/davidjspooner/net-mapper/pkg/snmp/mibtoken"
)

type patternSequence struct {
	valueBase
	pattern []ValueReader
}

var _ Type = (*patternSequence)(nil)

func (patternSequence *patternSequence) Source() mibtoken.Source {
	return patternSequence.source
}

func (patternSequence *patternSequence) readDefinition(ctx context.Context, module *Module, s mibtoken.Reader) error {
	for !s.IsEOF() {
		peek1, err := s.LookAhead(1)
		if err == nil && peek1.String() == "::=" {
			return nil
		}
		//read one peek set
		peek, err := s.LookAhead(0)
		if err != nil {
			return err
		}
		switch peek.Type() {
		case mibtoken.IDENT:
			ref := &TypeReference{
				ident: peek,
			}
			ref.set(module, nil, *peek.Source())
			s.Pop()
			if ref.ident.String() == "SEQUENCE OF" {
				ref.sequenceOf = true
				ref.ident, err = s.Pop()
				if err != nil {
					return err
				}
			}
			peek, err := s.LookAhead(0)
			if err == nil {
				close, ok := brackets[peek.String()]
				if ok {
					open := peek.String()
					constraint, err := mibtoken.ReadBlock(s, open, close)
					if err != nil {
						return err
					}
					ref.constraint = constraint
				}
			}
			patternSequence.pattern = append(patternSequence.pattern, ref)
		case mibtoken.STRING:
			unescape, err := mibtoken.Unquote(peek)
			if err != nil {
				return mibtoken.WrapError(s, err)
			}
			constTok := ExpectedToken{text: unescape}
			patternSequence.pattern = append(patternSequence.pattern, &constTok)
			s.Pop()
		case mibtoken.SYMBOL:
			peekTxt := peek.String()
			switch peekTxt {
			case "|":
				return nil
			default:
				return peek.WrapError(asn1error.NewUnexpectedError("IDENT", peek.String(), "token in patternSequence.read"))
			}
		}
	}
	return nil
}

func (patternSequence *patternSequence) isDefined() bool {
	return len(patternSequence.pattern) > 0
}

func (patternSequence *patternSequence) readValue(ctx context.Context, module *Module, s mibtoken.Reader) (Value, error) {

	ctx = patternSequence.valueBase.module.withContext(ctx)

	depth := getDepth(ctx)
	if depth.Inc(patternSequence) > 100 {
		return nil, patternSequence.source.Errorf("depth limit reached")
	}
	defer depth.Dec(patternSequence)

	lastName := ""

	tmp := mibtoken.NewProjection(s)
	composite := CompositeValue{
		fields: make(map[string]Value),
		vType:  patternSequence,
	}

	for i, pattern := range patternSequence.pattern {

		value, err := pattern.readValue(ctx, module, tmp)
		if err != nil {
			return nil, err
		}
		if value != nil {
			otherComposite, _ := value.(*CompositeValue)
			if otherComposite != nil {
				for k, v := range otherComposite.fields {
					composite.fields[k] = v
				}
			} else if lastName != "" {
				composite.fields[lastName] = value
			} else {
				composite.fields[fmt.Sprintf("%d", i)] = value
			}
		} else {
			expected, ok := pattern.(*ExpectedToken)
			if ok {
				c := expected.text[0]
				if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') {
					lastName = expected.text
				}
			}
		}
	}
	tmp.Commit()
	if len(composite.fields) == 0 && lastName != "" {
		v := &GoValue[string]{valueBase: patternSequence.valueBase, value: lastName}
		return v, nil
	}
	return &composite, nil
}

type patternChoice struct {
	valueBase
	alternatives []Type
}

var _ Type = (*patternChoice)(nil)

func (choice *patternChoice) readDefinition(ctx context.Context, module *Module, s mibtoken.Reader) error {
	for !s.IsEOF() {
		if len(choice.alternatives) > 0 {
			peek, _ := s.LookAhead(0)
			if peek.String() != "|" {
				break
			}
			s.Pop()
		}
		peek, err := s.LookAhead(0)
		if err != nil {
			return err
		}
		switch peek.String() {
		case "value":
			mibtoken.ReadExpected(s, "value")
			block, err := mibtoken.ReadBlock(s, "(", ")")
			if err != nil {
				return err
			}
			label, err := block.Pop()
			if err != nil {
				return err
			}
			typeName, err := block.Pop()
			if err != nil {
				typeName = label
			}
			simpleType := &TypeReference{ident: typeName}
			simpleType.set(module, nil, *label.Source())
			choice.alternatives = append(choice.alternatives, simpleType)
		case "type":
			mibtoken.ReadExpected(s, "type")
			block, err := mibtoken.ReadBlock(s, "(", ")")
			if err != nil {
				simpleType := &TypeReference{ident: peek}
				simpleType.set(module, nil, *peek.Source())
				choice.alternatives = append(choice.alternatives, simpleType)
				continue
			}
			typeName, err := block.Pop()
			if err != nil {
				return err
			}
			typeDef, _, err := Lookup[Type](ctx, typeName.String())
			if err != nil {
				return err
			}
			choice.alternatives = append(choice.alternatives, typeDef)
		default:
			alternative := &patternSequence{}
			alternative.set(module, nil, *peek.Source())
			err := alternative.readDefinition(ctx, module, s)
			if err != nil {
				return err
			}
			if !alternative.isDefined() {
				break
			}
			choice.alternatives = append(choice.alternatives, alternative)
		}
	}
	return nil
}

func (choice *patternChoice) Source() mibtoken.Source {
	return choice.source
}

func (choice *patternChoice) readValue(ctx context.Context, module *Module, s mibtoken.Reader) (Value, error) {

	errs := asn1error.List{}
	for _, alt := range choice.alternatives {
		tmp := mibtoken.NewProjection(s)
		value, err := alt.readValue(ctx, module, tmp)
		if err == nil {
			tmp.Commit()
			return value, nil
		}
		errs = append(errs, err)
	}
	switch len(errs) {
	case 1:
		return nil, errs[0]
	case 0:
		return nil, s.Source().Errorf("no choice matched")
	default:
		return nil, errs
	}
}
