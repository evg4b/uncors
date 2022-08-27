package pool

import (
	"context"
	"sync"

	"github.com/hashicorp/go-multierror"
)

type Pool struct {
	wg       sync.WaitGroup
	cancel   context.CancelFunc
	once     sync.Once
	errMutex sync.Mutex
	err      *multierror.Error
}

func WithContext(ctx context.Context) (*Pool, context.Context) {
	ctx, cancel := context.WithCancel(ctx)

	return &Pool{cancel: cancel}, ctx
}

func (g *Pool) Wait() error {
	g.wg.Wait()
	g.Cancel()

	return g.err.ErrorOrNil() // nolint: wrapcheck
}

func (g *Pool) Go(action func() error) {
	g.wg.Add(1)

	go func() {
		defer g.wg.Done()

		if err := action(); err != nil {
			g.errMutex.Lock()
			g.err = multierror.Append(g.err, err)
			g.errMutex.Unlock()
			g.once.Do(func() {
				g.Cancel()
			})
		}
	}()
}

func (g *Pool) Cancel() {
	if g.cancel != nil {
		g.cancel()
	}
}
