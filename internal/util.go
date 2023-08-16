package internal

import "github.com/samber/lo"

func Any[T any](collection []T, predicate func(item T) bool) bool {
	_, ok := lo.Find(collection, predicate)
	return ok
}

func Both[T any](collection []T, predicate func(item T) bool) bool {
	for _, t := range collection {
		if !predicate(t) {
			return false
		}
	}
	return true
}
