package gen

import (
	"fmt"
	"sort"
	"sync"
)

// OutputFile represents a single generated file.
type OutputFile struct {
	Path        string // Relative path within output directory
	Content     []byte
	Scaffold    bool // If true, only write when file doesn't already exist
	ProjectFile bool // If true, write to parent of output directory (for Makefiles, platform stubs)
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

// GeneratorsForImplLang returns the generator names for impl scaffolding, Makefile, and platform services.
func GeneratorsForImplLang(implLang string) []string {
	switch implLang {
	case "cpp":
		return []string{"impl_cpp", "impl_makefile_cpp", "impl_platform_services"}
	case "rust":
		return []string{"impl_rust", "impl_makefile_rust", "impl_platform_services"}
	case "go":
		return []string{"impl_go", "impl_makefile_go", "impl_platform_services"}
	case "c":
		return []string{"impl_makefile_c", "impl_platform_services"}
	default:
		return nil
	}
}

// GeneratorsForImplLangAndTargets returns extra generators needed for a specific
// impl_lang + targets combination. For Go with a web target, this returns impl_go_wasm
// so that //go:wasmexport scaffolding is generated alongside the cgo shim.
func GeneratorsForImplLangAndTargets(implLang string, targets []string) []string {
	if implLang == "go" {
		for _, t := range targets {
			if t == "web" {
				return []string{"impl_go_wasm"}
			}
		}
	}
	return nil
}
