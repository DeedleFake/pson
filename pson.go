// Package pson implements experimental support for progressive JSON.
package pson

import (
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"sync"
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

type marshalState struct {
	ctx context.Context
	c   chan result
	wg  sync.WaitGroup
	id  uint64
}

func (state *marshalState) Marshal(e *jsontext.Encoder, f AsyncFunc) error {
	id := fmt.Sprintf("$pson:%v", state.id)
	state.id++

	state.wg.Go(func() {
		done := make(chan struct{})
		val, err := f(state.ctx)
		select {
		case <-state.ctx.Done():
			return
		case state.c <- result{ID: id, Val: val, Err: err, Done: done}:
			<-done
		}
	})

	return e.WriteToken(jsontext.String(id))
}

func (state *marshalState) Wait() {
	state.wg.Wait()
	close(state.c)
}

func marshalChunk(out ChunkWriter, v any, opts ...json.Options) error {
	chunk, err := out.Chunk()
	if err != nil {
		return err
	}

	merr := json.MarshalWrite(chunk, v, opts...)
	cerr := chunk.Close()
	return errors.Join(merr, cerr)
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
// Every marshal, first for in and then for the results of the various
// AsyncFunc calls, will be written to out as a separate chunk. These
// will be obtained, written to, and then immediately closed. These
// calls do not happen concurrently with other chunks being written.
//
// Every call to an AsyncFunc is passed a context dervived from the
// provided ctx but that is canceled when Marshal returns.
//
// Marshal does not return until all AsyncFunc calls have fully exited
// or until one of them returns an error or an error is encountered
// during the encoding process.
func Marshal(ctx context.Context, out ChunkWriter, in any, opts ...json.Options) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	state := &marshalState{
		ctx: ctx,
		c:   make(chan result),
	}

	opt := json.JoinOptions(opts...)
	m, _ := json.GetOption(opt, json.WithMarshalers)
	m = json.JoinMarshalers(m, json.MarshalToFunc(state.Marshal))
	opt = json.JoinOptions(opt, json.WithMarshalers(m))

	err := marshalChunk(out, in, opt)
	if err != nil {
		return err
	}

	go state.Wait()
	for r := range state.c {
		if r.Err != nil {
			close(r.Done)
			return r.Err
		}

		err := marshalChunk(out, map[string]any{r.ID: r.Val}, opt)
		close(r.Done)
		if err != nil {
			return err
		}
	}

	return nil
}

// ChunkWriter is something that can be written to in pieces. This is
// a way for Marshal to make sure that data is sent after each piece
// of JSON is ready. For simple use-cases, see [Chunk].
type ChunkWriter interface {
	Chunk() (io.WriteCloser, error)
}

type chunkWriter struct {
	io.Writer
}

// Chunk returns a ChunkWriter that returns chunks that write directly
// to w and have no-op Close methods.
func Chunk(w io.Writer) ChunkWriter {
	return &chunkWriter{Writer: w}
}

func (w *chunkWriter) Chunk() (io.WriteCloser, error) {
	return w, nil
}

func (w *chunkWriter) Close() error {
	return nil
}
