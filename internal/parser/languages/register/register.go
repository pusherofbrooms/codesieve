package register

import (
	"fmt"
	"sync"

	"github.com/pusherofbrooms/codesieve/internal/parser/spec"
)

var (
	mu       sync.RWMutex
	parseFns = map[string]spec.ParseFunc{}
)

func MustRegister(name string, parse spec.ParseFunc) {
	if parse == nil {
		panic(fmt.Sprintf("parse function for %q is nil", name))
	}
	mu.Lock()
	defer mu.Unlock()
	if _, exists := parseFns[name]; exists {
		panic(fmt.Sprintf("parse function already registered for %q", name))
	}
	parseFns[name] = parse
}

func Lookup(name string) (spec.ParseFunc, bool) {
	mu.RLock()
	defer mu.RUnlock()
	fn, ok := parseFns[name]
	return fn, ok
}

func Names() []string {
	mu.RLock()
	defer mu.RUnlock()
	out := make([]string, 0, len(parseFns))
	for name := range parseFns {
		out = append(out, name)
	}
	return out
}
