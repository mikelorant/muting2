package format

import (
	"fmt"
	"strings"
)

func SliceToFormattedLines[T any](vs []T) string {
	var strs []string
	for _, v := range vs {
		strs = append(strs, fmt.Sprint(v))
	}
	return strings.Join(strs, "\n")
}

func SliceToFormattedLinesWithPrefix[T any](vs []T, prefix string) string {
	var strs []string
	for _, v := range vs {
		strs = append(strs, fmt.Sprintf("%v %v", prefix, v))
	}
	return strings.Join(strs, "\n")
}
