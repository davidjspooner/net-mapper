package token

type Match interface {
	Count() int
	ForEach(func(key string, value any) error) error
}

type MatchSequence []Match

func (ms MatchSequence) Count() int {
	count := 0
	for _, m := range ms {
		count += m.Count()
	}
	return count
}

func (ms MatchSequence) ForEach(f func(key string, value any) error) error {
	for _, m := range ms {
		if err := m.ForEach(f); err != nil {
			return err
		}
	}
	return nil
}

type MatchAlternatives []Match


