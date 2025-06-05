package pson_test

import (
	"context"
	"strings"
	"testing"

	"deedles.dev/pson"
	"github.com/stretchr/testify/require"
)

func TestMarshal(t *testing.T) {
	var buf strings.Builder
	err := pson.Marshal(t.Context(), pson.Chunk(&buf), map[string]any{
		"test": pson.AsyncFunc(func(ctx context.Context) (any, error) {
			return map[string]any{
				"recursive": pson.AsyncFunc(func(ctx context.Context) (any, error) {
					return 3, nil
				}),
			}, nil
		}),
	})
	require.Nil(t, err)
	require.Equal(t, `{"test":"$pson:0"}{"$pson:0":{"recursive":"$pson:1"}}{"$pson:1":3}`, buf.String())
}
