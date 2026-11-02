package repeat

var (
	alpha       = 1
	beta, gamma = 2, 3
)

func init() { alpha++ }

func init() { beta += gamma }
