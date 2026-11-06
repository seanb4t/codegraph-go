package res

import u "example.com/res/util"

func local() int { return 1 }

func Caller() int { return u.Helper() + local() + local() }

func Shadow() int {
	local := func() int { return 2 }
	return local()
}

func Recv() int {
	var t u.Thing
	return t.Run()
}
