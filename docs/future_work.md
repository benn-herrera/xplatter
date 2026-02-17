# Future Work

## RESOLVED - architecture discussion: requirements, complexity, value vs cost of adding Android + Kotlin-KMP to Android/Kotlin and the rest as a code gen target
  * RESOLUTION:
    * adding an Android/KMP and maybe iOS/KMP binding target has value
    * great candidate for v2. get through v1 first (or better yet, get someone to pay for it)


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