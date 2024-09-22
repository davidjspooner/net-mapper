package token

import "fmt"

type Pattern[T any] interface {
	Match(List[T]) (MatchAlternatives, error)
}

type PatternSequence[T any] []Pattern[T]

func (tpl PatternSequence[T]) Match(tokens List[T]) (MatchAlternatives, error) {
	if len(tpl) == 0 {
		return nil, nil
	}

	matchalternatives, err := tpl[0].Match(tokens)
	if err != nil {
		return nil, err
	}
	if len(tpl) == 1 {
		return matchalternatives, nil
	}

	result := matchalternatives

	for _, matchAlternative := range matchalternatives {
		matchSequence, ok := matchAlternative.(MatchSequence)
		if !ok {
			matchSequence = MatchSequence{matchAlternative}
		}
		tail := tokens[matchAlternative.Count():]
		submatches, err := tpl[1:].Match(tail)
		if err != nil {
			return nil, err
		}
		for _, submatch := range submatches {
			result = append(result, append(matchSequence, submatch))
		}
	}
	return matchalternatives, nil
}

type PatternAlternates[T any] []Pattern[T]

func (tpa PatternAlternates[T]) Match(tokens List[T]) (MatchAlternatives, error) {
	result := MatchAlternatives{}
	for _, tp := range tpa {
		matches, err := tp.Match(tokens)
		if err != nil {
			return nil, err
		}
		for _, match := range matches {
			result = append(result, match)
		}
	}
	return result, nil
}

type PatternRepeat[T any] struct {
	Element   Pattern[T]
	Delimiter Pattern[T]
}

func (tpr PatternRepeat[T]) Match(tokens List[T]) (MatchAlternatives, error) {
	//TBD
	return nil, fmt.Errorf("not implemented - PatternRepeat.Match")
}

type BlockPattern[T any] struct {
	Open  Pattern[T]
	Close Pattern[T]
}

func (bp BlockPattern[T]) Match(tokens List[T]) (MatchAlternatives, error) {

	//openAlternates, err := bp.Open.Match(tokens)
	//TBD
	return nil, fmt.Errorf("not implemented - PatternRepeat.Match")
}
