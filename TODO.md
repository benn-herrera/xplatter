# TODO

## architecture discussion: requirements, complexity, value vs cost of adding Android + Kotlin-KMP to Android/Kotlin and the rest as a code gen target

## architectue discussion: wasm package production strategy
  * let's have an architecture discussion on the tradeoffs of using the language-specific binding mechanisms to wasm (e.g. emscripten for c++) for cpp, rust, go instead of routing all languages through a pure C ABI.
    * it does require code gen for 3 completely different binding systems (complexity)
    * it does leverage those existing binding systems capacity for generating idiomatic bindings
    * it makes WASM a 'special stepchild' in the target binding space
    * WASM already has 'special stepchild' issues, it's just a matter of minimizing the pain.
    * the pure C api path still allows users choosing unusual system languages to implement under the generated header with their own export mechanism (an agnostic path is preserved)
  * the ultimate goal of the project is to alleviate a significant pain point for developers of high performance, all-platform code.
    * if we have to eat the ugly but the consumers of this project get a clean experience on their projects they won't care.
    * we want to minimize the amount of ugly (maintenance overhead, corner case failure risks) we eat to allow for reliable behavior and updates.

## RESOLVED - architecture discussion: value of offering go lang as a highly supported implementation language
  * Hypothesis: preserving Go as a suported implementation language for all-platform APIs has distinct value. Premises:
    * allows faster prototyping and alpha delivery for product validation
      * agents code more reliably in Go than C++ or even Rust
      * Go has less exploratory iteration friction than Rust and far less complexity than C++ for an agentic coding system to navigate
      * Go has a much simpler build system and 3rd party dependency ecology than C++ (though probably close to par with Rust)
    * the system allows for transparently swapping implementation languages under the api for beta and final releases
      * generating C++ or Rust from an accumulated, maintained spec plus a working reference implementation in Go at the end of a project's exploratory phase can be done reliably
      * most projects will see a significant net savings in tokens, bugs, time, and risk for the approach of prototyping in Go and finishing in C++ or Rust
  * provide the pros and cons of the hypothesis and its premises and provide a 0-9 value where 0 is that the hypothesis not supportable at all and 9 is that it is nearly irrefutable for each premise and the hypothesis as a whole
  * RESOLUTION:
    * the hypothesis and its premises are defensible, but gut-based without empirical data
    * doing multiple case studies would be expensive
    * the path to supporting go is reasonably achievable
    * the premise of utility can be put for as an expert opinion in the README
    * the reality of it can be evaluated in the follow-on demo project that will consume the xplattergy capability.