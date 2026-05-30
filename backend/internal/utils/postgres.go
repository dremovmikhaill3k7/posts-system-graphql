package utils

import "fmt"

func CreatePlaceholders(ids []int) (placeholders []string, args []any) {
	placeholders = make([]string, len(ids))
	args = make([]any, len(ids))
	for i, id := range ids {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	return placeholders, args
}
