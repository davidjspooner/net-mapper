package asn1mib

import (
	"fmt"
	"os"
)

type Import struct {
	Identifiers []string
	From        string
}

type ImportList []Import

type file struct {
	Name    string
	Imports ImportList
	Tokens  []string
}

func (mf *file) ID() string {
	return mf.Name
}

func (mf *file) readImports(s *Scanner) error {
	var err error
	for s.Scan() {
		imp := Import{}
		tokenType, tok := s.Token()
		if tok == ";" {
			return nil
		}
		if tokenType == IDENT {
			imp.Identifiers = append(imp.Identifiers, tok)
		} else {
			return fmt.Errorf("unexpected token: %s", tok)
		}
		for s.Scan() {
			tokenType, tok = s.Token()
			if tok == "," {
				tok, err = s.ScanIdent()
				if err != nil {
					return err
				}
				imp.Identifiers = append(imp.Identifiers, tok)
			} else if tok == "FROM" {
				tok, err = s.ScanIdent()
				if err != nil {
					return err
				}
				imp.From = tok
				mf.Imports = append(mf.Imports, imp)
				break
			} else {
				return fmt.Errorf("unexpected %s: %s", tokenType, tok)
			}
		}
		if s.Err() != nil {
			return s.Err()
		}

	}
	return s.Err()
}

func (mf *file) readFile(s *Scanner) error {

	var err error
	mf.Name, err = s.ScanIdent()
	if err != nil {
		return err
	}

	err = s.ScanAndExpect("DEFINITIONS", "::=", "BEGIN")
	if err != nil {
		return err
	}

	if !s.Scan() {
		return s.Err()
	}
	tokenType, tok := s.Token()

	if tokenType == IDENT && tok == "IMPORTS" {
		err = mf.readImports(s)
		if err != nil {
			return err
		}
	} else {
		mf.Tokens = append(mf.Tokens, tok)
	}

	for s.Scan() {
		_, tok = s.Token()
		if tok == "IDENTIFIER" && len(mf.Tokens) > 0 && mf.Tokens[len(mf.Tokens)-1] == "OBJECT" {
			mf.Tokens[len(mf.Tokens)-1] = "OBJECT_IDENTIFIER"
		} else {
			mf.Tokens = append(mf.Tokens, tok)
		}
	}

	if err != nil {
		return err
	}

	return err
}

func newFile(filename string) (*file, error) {
	mf := &file{}
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	s, err := NewScanner(f, WithFilename(filename), WithSkip(WHITESPACE, COMMENT))
	if err != nil {
		return nil, err
	}
	err = mf.readFile(s)
	if err != nil {
		return nil, err
	}
	return mf, nil
}
