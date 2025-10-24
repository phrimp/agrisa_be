package worker

import (
	"context"
	"sync"
)

type (
	Job    func()
	Worker interface {
		Run(ctx context.Context, wg *sync.WaitGroup)
	}
)

type Pool interface {
	Start(ctx context.Context, managerWg *sync.WaitGroup)
}
