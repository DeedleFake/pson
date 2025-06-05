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
	err := pson.Marshal(t.Context(), &buf, map[string]any{
		"test": pson.AsyncFunc(func(ctx context.Context) (any, error) {
			return map[string]any{
				"recursive": pson.AsyncFunc(func(ctx context.Context) (any, error) {
					return 3, nil
				}),
			}, nil
		}),
	})
	require.Nil(t, err)
	require.Equal(t, `{"test":"$pson:1"}{"$pson:1":{"recursive":"$pson:2"}}{"$pson:2":3}`, buf.String())
}
