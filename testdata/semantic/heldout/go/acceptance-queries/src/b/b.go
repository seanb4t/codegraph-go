package b

import "example.com/hq/a"

func Commit() int { return a.Save() }

func Rollback() int { return a.Audit() }
