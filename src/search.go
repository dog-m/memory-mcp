package main

import (
	"fmt"
	"regexp"
	"regexp/syntax"

	"github.com/redmatter/go-globre/v2"
)

type SearchMatcher = func(string) bool

func getTextMatcher(pattern *string) (SearchMatcher, error) {
	// note: any number of consecutive "*", ie. "**", are treated like a single one, thus no filtering required
	r_patt := globre.RegexFromGlob(
		*pattern,
		globre.ExtendedSyntaxEnabled(true), // allow "[0-9]", "{apple,banana}" and "?"
		globre.GlobStarEnabled(false),      // simplify the function of "*" as just "anything" (same as "**")
		globre.WithDelimiter('.'),          // split using basic punctuation (ignored)
	)

	// make patterns suitable for "whole-text" operation by cutting-off "^" and "$"
	r_patt = r_patt[1 : len(r_patt)-1]

	// build
	if r, err := regexp.Compile(r_patt); err != nil {
		return nil, err
	} else {
		return r.MatchString, nil
	}
}

func formatParserError(err error) error {
	if e, ok := err.(*syntax.Error); !ok || e == nil {
		panic(fmt.Sprintf("Unexpected error object type: %T", err))
	} else {
		switch e.Code {
		case syntax.ErrInternalError:
			return fmt.Errorf("unknown internal parser error")
		case syntax.ErrInvalidPerlOp:
			return fmt.Errorf("invalid or unsupported syntax")
		case syntax.ErrMissingBracket, syntax.ErrMissingParen:
			return fmt.Errorf("missing closing bracket")
		case syntax.ErrUnexpectedParen:
			return fmt.Errorf("unexpected open bracket")
		default:
			return fmt.Errorf("%v", e.Code)
		}
	}
}

var MATCH_ALL SearchMatcher = func(string) bool {
	return true
}
var MATCH_NONE SearchMatcher = func(string) bool {
	return false
}

func GetSearchMatchers(include *string, exclude *string) (SearchMatcher, SearchMatcher, error) {
	var allow, deny SearchMatcher

	if include == nil {
		allow = MATCH_ALL
	} else {
		if matcher, err := getTextMatcher(include); err != nil {
			return nil, nil, fmt.Errorf("Invalid INCLUDE pattern: %v", formatParserError(err))
		} else {
			allow = matcher
		}
	}

	if exclude == nil {
		deny = MATCH_NONE
	} else {
		if matcher, err := getTextMatcher(exclude); err != nil {
			return nil, nil, fmt.Errorf("Invalid EXCLUDE pattern: %v", formatParserError(err))
		} else {
			deny = matcher
		}
	}

	return allow, deny, nil
}
