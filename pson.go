package pson

import (
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
)

type AsyncFunc func(ctx context.Context) (any, error)

type result struct {
	ID   string
	Val  any
	Err  error
	Done chan struct{}
}

func Marshal(ctx context.Context, out io.Writer, in any, opts ...json.Options) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	opt := json.JoinOptions(opts...)
	m, _ := json.GetOption(opt, json.WithMarshalers)

	c := make(chan result)
	var wg sync.WaitGroup
	var n uint64

	m = json.JoinMarshalers(m, json.MarshalToFunc(func(e *jsontext.Encoder, f AsyncFunc) error {
		id := fmt.Sprintf("$pson:%v", atomic.AddUint64(&n, 1))
		wg.Go(func() {
			done := make(chan struct{})
			val, err := f(ctx)
			select {
			case <-ctx.Done():
				return
			case c <- result{ID: id, Val: val, Err: err, Done: done}:
				<-done
			}
		})

		return e.WriteToken(jsontext.String(id))
	}))

	opt = json.JoinOptions(opt, json.WithMarshalers(m))
	err := json.MarshalWrite(out, in, opt)
	if err != nil {
		return err
	}

	go func() {
		wg.Wait()
		close(c)
	}()

	for r := range c {
		if r.Err != nil {
			close(r.Done)
			return r.Err
		}
		err := json.MarshalWrite(out, map[string]any{r.ID: r.Val}, opt)
		close(r.Done)
		if err != nil {
			return err
		}
	}

	return nil
}
