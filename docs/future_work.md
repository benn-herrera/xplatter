# Future Work

## V2 features (or good for-pay candidates):
  * add an Android/KMP and maybe iOS/KMP binding targets
  * support exporting a typescript binding instead of javascript for web targets.
  * add front door support for zig as an implementation language
  * a/b implementation swapping? ability to select between two implementations at startup time?
    * would require providing 2 dynamic impl libraries (different backing langs or diff versions from same lang) and start time selection of the dynamic library.
    * worth investigating, but v1 has enough on its plate
  * generate type names, member names, function names in target language casing conventions (e.g. PascalCase, camelCase, snake_case, etc.)

## Keep an eye on the cost/value of Go as a supported implementation language
  * decision to include it is gut level backed by some experience
    * COULD offer quick, agentic driven prototyping & discovery with higher reliability
    * with working Go impl + agentically maintained spec porting from discovery language (Go) to delivery language (Rust, C, or C++) will be cheaper than ever before.
    * COULD be an attactive nuisance
  * experiment during building of projects downstream of xplatter
  * if it shows deal breaker issues or just doesn't pan out cut it and save the hassle.

## Add additional use cases to READMEs

* Rapid AI-assisted development from prototype through alpha in fast/loose language (Zig, Go) with minimal iterative friction. Final alpha version is low-cost, AI executed port from fast/loose language to safe language (e.g. Rust) aided by full spec (AI maintained during development) and working reference implementation.
  * Build & discover fast and throw the first one away has gone from 'ideal for engineering, impractical for business' to 'best of all worlds'.
    * The product ready implementation can be transparently swapped under the front end code.
    * 'Build fast' was the point of agreement. 'Rebuild it right' was always the point of conflict due to cost.
    * Now the cost is low and the benefit is clear: provable safety
* Expanding reach of existing system language libraries
  * Use AI to infer the yaml/fbs API spec from existing code and
    * Replace authored impl language API defining files with generated
    * OR have implementation be thin wrapper around existing API
