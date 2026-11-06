package api

import "example.com/q/store"

func Handle() int { return store.Load() }

func Unrelated() int { return 3 }
