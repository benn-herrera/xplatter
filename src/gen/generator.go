package gen

import (
	"fmt"
	"sort"
	"sync"
)

// OutputFile represents a single generated file.
type OutputFile struct {
	Path    string // Relative path within output directory
	Content []byte
}

// Generator is the interface all code generators implement.
// Each generator produces output files for a specific target (e.g., C header, Kotlin/JNI, Swift).
// Adding a new language requires only implementing this interface and calling Register() in init().
type Generator interface {
	// Name returns the generator name (e.g., "cheader", "kotlin", "swift").
	Name() string

	// Generate produces output files for the given API definition.
	Generate(ctx *Context) ([]*OutputFile, error)
}

var (
	registryMu sync.RWMutex
	registry   = map[string]func() Generator{}
)

// Register adds a generator factory to the registry.
// Typically called from init() in each generator's file.
func Register(name string, factory func() Generator) {
	registryMu.Lock()
	defer registryMu.Unlock()
	if _, exists := registry[name]; exists {
		panic(fmt.Sprintf("generator %q already registered", name))
	}
	registry[name] = factory
}

// Get returns a new instance of the named generator.
func Get(name string) (Generator, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	factory, ok := registry[name]
	if !ok {
		return nil, false
	}
	return factory(), true
}

// All returns the names of all registered generators, sorted.
func All() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// GeneratorsForTarget returns the generator names needed for a given target platform.
func GeneratorsForTarget(target string) []string {
	switch target {
	case "android":
		return []string{"kotlin"}
	case "ios", "macos":
		return []string{"swift"}
	case "web":
		return []string{"jswasm"}
	case "windows", "linux":
		// Desktop targets use the C header directly
		return nil
	default:
		return nil
	}
}

// GeneratorsForImplLang returns the generator name for impl scaffolding.
func GeneratorsForImplLang(implLang string) string {
	switch implLang {
	case "cpp":
		return "impl_cpp"
	case "rust":
		return "impl_rust"
	case "go":
		return "impl_go"
	case "c":
		return "" // No scaffolding for pure C
	default:
		return ""
	}
}
