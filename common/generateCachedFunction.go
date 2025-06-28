package common

import (
	"fmt"
	"hash/fnv"
	"reflect"
	"time"

	"github.com/ornovog/cache/evictions"
)

var (
	// Cache capacity and TTL for demonstration
	commonTTL  = 5 * time.Minute
	maxEntries = 1000
)

func generateKey(args ...any) string {
	h := fnv.New64a()
	for _, arg := range args {
		h.Write([]byte(fmt.Sprintf("%v|", arg)))
	}
	return fmt.Sprintf("%x", h.Sum64())
}

func NewCachedFunction[F interface{}](fn F) F {
	valFn := reflect.ValueOf(fn)
	typeFn := valFn.Type()
	dedupe := NewInFlightDedup[any]()
	store := NewStorage[any](commonTTL, maxEntries, evictions.NewLRUPolicy())

	wrapped := reflect.MakeFunc(typeFn, func(args []reflect.Value) []reflect.Value {
		keyParts := make([]any, len(args))
		for i, v := range args {
			keyParts[i] = v.Interface()
		}
		key := generateKey(keyParts...)
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

		store.Set(key, val, err)
		dedupe.Finish(key, val, err)

		return out
	})

	return wrapped.Interface().(F)
}
