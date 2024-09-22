package asn1mib

import "strings"

type TokenList []Token

func (tl TokenList) String() string {
	sb := &strings.Builder{}
	for n, t := range tl {
		if n > 0 {
			sb.WriteString(" ")
		}
		sb.WriteString(t.String())
	}
	return sb.String()
}

func (tl *TokenList) DeleteIndex(n int) {
	copy((*tl)[n:], (*tl)[n+1:])
	*tl = (*tl)[:len(*tl)-1]
}

func (tl *TokenList) RemoveHead() {
	if len(*tl) > 0 {
		tl.DeleteIndex(0)
	}
}
