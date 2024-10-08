package dzi

import "regexp"

var mathRegExp *regexp.Regexp

func init() {
	regExp, err := regexp.Compile(`\((.*)\)`)
	if err != nil {
		panic(err)
	}
	mathRegExp = regExp
}

func matchSwatch(in string) string {
	match := mathRegExp.FindStringSubmatch(in)
	if match == nil {
		return ""
	}
	return match[1]
}
