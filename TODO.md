# TODO

## RESOLVED - architectue discussion: wasm package production strategy
  * let's have an architecture discussion on the tradeoffs of using the language-specific binding mechanisms to wasm (e.g. emscripten for c++) for cpp, rust, go instead of routing all languages through a pure C ABI.
    * it does require code gen for 3 completely different binding systems (complexity)
    * it does leverage those existing binding systems capacity for generating idiomatic bindings
    * it makes WASM a 'special stepchild' in the target binding space
    * WASM already has 'special stepchild' issues, it's just a matter of minimizing the pain.
    * the pure C api path still allows users choosing unusual system languages to implement under the generated header with their own export mechanism (an agnostic path is preserved)
  * the ultimate goal of the project is to alleviate a significant pain point for developers of high performance, all-platform code.
    * if we have to eat the ugly but the consumers of this project get a clean experience on their projects they won't care.
    * we want to minimize the amount of ugly (maintenance overhead, corner case failure risks) we eat to allow for reliable behavior and updates.
  * RESOLUTION: //go:wasmexport to produce an additional binding output for go->WASM. preserve the C ABI binding for other platforms.
  
## Execute on the resolved conversation above
* Add Go -> WASM support via //go:wasmexport (right now go -> WASM is broken)
* leave the existing go -> C ABI exposure for other platform/language targets
