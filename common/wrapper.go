package common

import (
	"fmt"
	"hash/fnv"
	"reflect"
)

func GenerateKey(args ...any) string {
	h := fnv.New64a()
	for _, arg := range args {
		h.Write([]byte(fmt.Sprintf("%v|", arg)))
	}
	return fmt.Sprintf("%x", h.Sum64())
}

func NewCachedFunction[F any, T any](fn F, store Storage[T]) F {
	valFn := reflect.ValueOf(fn)
	typeFn := valFn.Type()
	dedupe := NewInFlightDedup[any]()

	wrapped := reflect.MakeFunc(typeFn, func(args []reflect.Value) []reflect.Value {
		keyParts := make([]any, len(args))
		for i, v := range args {
			keyParts[i] = v.Interface()
		}
		key := GenerateKey(keyParts...)

		if val, err, ok := store.Get(key); ok {
			out := []reflect.Value{reflect.ValueOf(val)}
			if typeFn.NumOut() == 2 {
				out = append(out, reflect.Zero(typeFn.Out(1)))
				if err != nil {
					out[1] = reflect.ValueOf(err)
				}
			}
			return out
		}

		if val, err, ok := dedupe.Wait(key); ok {
			out := []reflect.Value{reflect.ValueOf(val)}
			if typeFn.NumOut() == 2 {
				out = append(out, reflect.Zero(typeFn.Out(1)))
				if err != nil {
					out[1] = reflect.ValueOf(err)
				}
			}
			return out
		}

		out := valFn.Call(args)

		var val any
		var err error
		if len(out) > 0 {
			val = out[0].Interface()
		}
		if len(out) > 1 && !out[1].IsNil() {
			err = out[1].Interface().(error)
		}

		store.Set(key, val.(T), err)
		dedupe.Finish(key, val, err)

		return out
	})

	return wrapped.Interface().(F)
}
