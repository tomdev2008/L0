package mongodb

import (
	"strings"
)

func isFind(str string) bool {
	return strings.Contains(str, "find")
}

func isdb(str string) bool {
	if str != "db" {
		return false
	}
	return true
}

func isParenthesesExist(str string) bool {
	return strings.Count(str, "(") == 1 && strings.Count(str, ")") == 1 && str[len(str)-1] == ')'
}
