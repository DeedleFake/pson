package main

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"deedles.dev/pson"
)

func main() {
	http.HandleFunc("/", func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(http.StatusOK)

		err := pson.Marshal(req.Context(), rw, map[string]any{
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
