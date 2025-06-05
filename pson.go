package pson

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"io"
)

type AsyncFunc func() (any, error)

func Marshal(out io.Writer, in any, opts ...json.Options) error {
	opt := json.JoinOptions(opts...)
	m, _ := json.GetOption(opt, json.WithMarshalers)

	var async []AsyncFunc
	m = json.JoinMarshalers(m, json.MarshalToFunc(func(e *jsontext.Encoder, f AsyncFunc) error {
		id := fmt.Sprintf("$pson:%v", len(async))
		async = append(async, func() (any, error) {
			v, err := f()
			if err != nil {
				return nil, err
			}
			return map[string]any{
				id: v,
			}, nil
		})

		return e.WriteToken(jsontext.String(id))
	}))

	opt = json.JoinOptions(opt, json.WithMarshalers(m))
	err := json.MarshalWrite(out, in, opt)
	if err != nil {
		return err
	}

	// It has to loop like this because the length of the slice can
	// change out from under it.
	for i := 0; i < len(async); i++ {
		f := async[i]

		v, err := f()
		if err != nil {
			return err
		}
		err = json.MarshalWrite(out, v, opt)
		if err != nil {
			return err
		}
	}

	return nil
}
