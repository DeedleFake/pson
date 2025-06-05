// Package pson implements experimental support for progressive JSON.
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

// An AsyncFunc represents a piece of data that should be sent later.
// See [Marshal] for more information.
type AsyncFunc func(ctx context.Context) (any, error)

type result struct {
	ID   string
	Val  any
	Err  error
	Done chan struct{}
}

// Marshal encodes a piece of data as JSON and writes it to out. This
// works almost exactly like [json.Marshal] except that it adds
// specialized support for [AsyncFunc] values. When one is encountered
// during the marshaling process, it is called concurrently. Once it
// returns and after the original object that it was encountered in
// has finished being marshaled, the value it returns is sent to out
// as well. This is performed recursively, allowing an AsyncFunc to
// return a value containing more AsyncFunc values which will in turn
// all be called in a similar way.
//
// Every call to an AsyncFunc is passed a context dervived from the
// provided ctx but that is canceled when Marshal returns.
//
// Marshal does not return until all AsyncFunc calls have fully exited
// or until one of them returns an error or an error is encountered
// during the encoding process.
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
