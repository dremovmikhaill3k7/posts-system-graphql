package graph

const (
	defaultLimit  = 20
	maxLimit      = 100
	defaultOffset = 0
)

func limitOffset(limit, offset *int) (int, int) {
	l := defaultLimit
	if limit != nil && *limit > 0 {
		l = *limit
		if l > maxLimit {
			l = maxLimit
		}
	}
	o := defaultOffset
	if offset != nil && *offset >= 0 {
		o = *offset
	}
	return l, o
}
