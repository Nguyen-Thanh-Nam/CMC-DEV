package scanner

import "context"

type Scanner interface {
	Scan(ctx context.Context, target string) (any, error)
}
