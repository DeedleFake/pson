package main

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"time"

	"deedles.dev/pson"
)

type chunkWriter struct {
	http.ResponseWriter
	*http.ResponseController
}

func newChunkWriter(rw http.ResponseWriter) pson.ChunkWriter {
	return &chunkWriter{
		ResponseWriter:     rw,
		ResponseController: http.NewResponseController(rw),
	}
}

func (w *chunkWriter) Chunk() (io.WriteCloser, error) {
	return w, nil
}

func (w *chunkWriter) Close() error {
	_, werr := w.Write([]byte{'\n'})
	ferr := w.Flush()
	return errors.Join(werr, ferr)
}

func main() {
	http.HandleFunc("/", func(rw http.ResponseWriter, req *http.Request) {
		cw := newChunkWriter(rw)

		err := pson.Marshal(req.Context(), cw, map[string]any{
			"test": pson.AsyncFunc(func(ctx context.Context) (any, error) {
				return map[string]any{
					"recursive": pson.AsyncFunc(func(ctx context.Context) (any, error) {
						time.Sleep(5 * time.Second)
						return 3, nil
					}),
				}, nil
			}),
		})
		if err != nil {
			slog.Error("failed to marshal", "err", err)
			return
		}
	})
	panic(http.ListenAndServe(":8080", nil))
}
