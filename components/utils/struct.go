package utils

import (
	"reflect"
	"unicode"
	"unicode/utf8"
)

func isExported(id string) bool {
	r, _ := utf8.DecodeRuneInString(id)
	return unicode.IsUpper(r)
}

// IsExported reports whether the struct filed is exported.
func IsExported(sf reflect.StructField) bool {
	if sf.Anonymous {
		t := sf.Type
		if t.Kind() == reflect.Ptr {
			t = t.Elem()
		}
		// If embedded, StructField.PkgPath is not a reliable
		// indicator of whether the field is exported.
		// See https://golang.org/issue/21122
		if !isExported(t.Name()) && t.Kind() != reflect.Struct {
			// Embedded fields of unexported non-struct types.
			// Do not ignore embedded fields of unexported struct types
			// since they may have exported fields.
			return false
		}
	} else if sf.PkgPath != "" {
		// Unexported non-embedded fields.
		return false
	}
	return true
}
