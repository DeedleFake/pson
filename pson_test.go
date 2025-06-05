package pson_test

import (
	"strings"
	"testing"

	"deedles.dev/pson"
	"github.com/stretchr/testify/require"
)

func TestMarshal(t *testing.T) {
	var buf strings.Builder
	err := pson.Marshal(&buf, map[string]any{
		"test": pson.AsyncFunc(func() (any, error) {
			return map[string]any{
				"recursive": pson.AsyncFunc(func() (any, error) {
					return 3, nil
				}),
			}, nil
		}),
	})
	require.Nil(t, err)
	t.Log(buf.String())
}
