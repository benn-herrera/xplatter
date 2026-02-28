*28 February 2026*
# The Splatting of Xplatter: Why I Tombstoned a Mobile Cross-Platform Binding Generator Halfway Through Building It

## The Promising Beginning
I started this project with a technical mindset to solve a problem that has recurred in my career. It wasn't because I loved this problem. I was actually sick of it and just wanted it solved so I could use the solution for other projects that would actually be fun. 

The problem in question is that high-performance native code libraries are difficult to share across mobile platforms, especially if you throw mobile web's WASM/JavaScript into the mix. Each platform has its own natural app language, and binding native libraries to them is a non-trivial hassle. The bindings also need to be performant. Routing touch events must have microsecond scale overhead to keep user interaction smooth. Maintaining bindings as your API evolves is a continuous drain in terms of time and bug risk. In short, it really sucks.

The solution started with a solid technical insight: Defining an API once in a standard data language instead of a coding language would allow for complete comprehension of the spec with a simple parser and without having to figure out how to translate idiomatic constructs from a source programming language to the target languages. It also meant that the generator could produce the C ABI that would tie the pieces together. Any language that can export to C was available to the developer for writing their one implementation. This would make it straightforward (finicky and verbose, but straightforward) to generate all of the stuff that it sucks to maintain while providing maximum flexibility.

Credit must be given to the Khronos Vulkan project. They use XML to define their API. Its value was clear, but their approach wasn't an exact fit. I absolutely stole it, but I took it to my chop shop before driving it around.

So I arrived at a design for maximal cross platform developer convenience:
* The user defines a single source of truth for their API
  * Schema validated JSON for the interface & function definitions
  * Flatbuffer files for the static data definitions
    * Mature, outstanding solution for efficient cross-language definitions of binary identical data structures
* The generation system produces
  * The C ABI contract, abstract interface definition, and exposure scaffolding for the chosen implementation language
  * The glue and binding code for the target languages
  * Idiomatic target language API on top of the bindings
  * Build projects for the binding layers that produce import-ready packages for each of the target/language combos
* The user implements once under the generated abstract API in their language of choice
* Propagation of API changes across all platforms becomes easy
  * Modify the API definition - the single source of truth
  * Build system dependencies generate matching binding changes
  * Update the single implementation

The target platforms would include not only Android/Kotlin, iOS/Swift, WASM/JavaScript, but also Linux/C++, Windows/C++, MacOS/C++ and MacOS/Swift. The desktop targets were easy to add, and as anyone who's ever done systems language coding on mobile will tell you, it is much, much faster and easier to iterate on a native code project on the host development system.

After a (clearly short) bit of noodling I landed on the name 'xplatter'. I was feeling really happy about the design at this point. If great artists steal, then I must be a damn genius!

## Getting the Show on the Road
In order to get started I had some decisions about tools and languages to make. I'd toyed with AI coding systems but hadn't really used one to do anything substantial. Claude Code had some rave reviews so I tried it on writing an [NYT Letterboxed](https://www.nytimes.com/puzzles/letter-boxed) puzzle solver, a test I've used in interviews and to evaluate new languages. Claude Code was impressive, so I decided that xplatter would be a good first real use case. I created some specialist agent definitions for the different platforms (Android, iOS, Web, macOS, Linux Windows) and disciplines (architecture review, tech writing).

After a session with the architecture agent I landed on some specifics. The generator tool itself would be coded in Go. It's fast, powerful, mature, good syntax, easy to deliver with, and the GC overhead would be a non-issue. I changed the API definition language from JSON to TOML - it's much more human-friendly. Lastly, I decided on the implementation languages to support for v1: pure C, C++, Rust, and Go. Zig would be easy to add later and even before that Zig's C interop is so clean it would be 3/4 supported already. This would provide a system where any one of C/C++, Rust, or Go could provide import ready packages for all of iOS/Swift, Android/Kotlin, WASM/JavaScript, macOS/Swift, desktop/C++.

I got to work with Claude Code and started building. I was assiduous about maintaining an AGENTS.md and an ARCHITECTURE.md to keep the development coherent and this paid off across fresh agent restarts. All in all it was great experience in learning how to work with high-autonomy coding agents and their various strengths and weaknesses.

## Half Way There With Home In Sight
I got far down the road to completion. I had a system that would build a distribution package that included the tool and had a hello-xplatter example project with a complete matrix of implementation languages and target platforms. With one make command you could get working Android, iOS, WASM, and desktop greeter applications that ran on their respective platforms. There were implementations of the greeter API in pure C, C++, Rust, and Go. The example project build system could mix and match any subset of target platform/language pairs to any implementation, and it was working on macOS, Linux, and Windows dev machines.

I call this the halfway point because I still needed to create a non-trivial demo project to consume xplatter. This was not only to prove it out, but to refine xplatter itself - specifically the performance/convenience tradeoffs in the generated bindings. I also needed a demo where I used AI to infer a xplatter API definition from an existing project to extend its reach.

I was still feeling pretty sanguine about the project. There was a lot left to do but it was more roadtrip than scavenger hunt. I was well on the way. Right arm!

It was around this time that I started doing some reflecting on the pain points from various projects professional and personal that had used manual solutions for the problem xplatter was addressing. Not all of the pain was from maintaining sync between the evolving native API and the bindings.

## The Rev-duh-lation

There was another recurring pain point. It was the repeatedly shifting ground under the feet of the platform-specific build processes in the form of tooling changes, API changes, and platform requirement changes. 

That's when it dawned on me. The root problems aren't technical and haven't been for a decade. There's a reason this problem still exists and it isn't because no one cared enough to solve it. It's because a durable solution has been actively subverted by first party platform providers. There's a war between the regulators & developers who want clear delivery paths and the first party platform providers fighting to protect massive revenue streams and developer lock-in. This project was tech savvy and business naive.

If I kept on, here's what would happen: I'd get to xplatter v0.5 and start on the interesting project that consumed it. I'd be toodling along making and testing my fun thing, and within a month one of the platform/language target packages would stop working. It would fail to build with a new error about not having frobnitzed the sekurenplatz. Or the built application would fail to run because I hadn't signed the shared library left handed in front of an autographed photo of Alan Turing. I'd spend a day or a week figuring out how the hell to change xplatter so it would generate code and build rules that jumped over the latest flaming shark tank. Then I'd try to pick up where I'd left off on the project I actually wanted to be working on. And then it would happen again on the other platform. And then again. And again.

I wasn't a genius stealing a good idea and running with it. I was an idiot building a greenhouse in the no man's land of a trench war. I was a masochist putting a hatchery in front of an autocannon fired by a motion sensor.

## The Mature Conclusion

The problem that xplatter tries to solve can't be durably addressed through technology alone. One of the most difficult things to learn as an engineer is to recognize a dead end. To eat the sunk cost and stop pouring effort into a flawed solution, however compelling. There's a [famous scene in the movie War Games](https://www.youtube.com/watch?v=1vmnp7ghGPk) where Dr. Falken explains the value of recognizing futility. For now, until the situation changes, trying to finish xplatter is tic-tac-toe. Stopping this project isn't quitting, it's the right decision.

The experience from working on it has been valuable, and the learning won't go to waste. Given that there's no viable way to finish xplatter at this time, the most productive next step is to tombstone it and write about the experience, which brings us here. There was nothing wrong with the greenhouse. It's just that no man's land is a terrible building site. For now, the right move is to leave the panes on the truck and build something different somewhere else.

## Epilogue: The Changing Legal and Regulatory Environment
There's a long-running [DOJ antitrust case against Apple](https://en.wikipedia.org/wiki/United_States_v._Apple_(2024)) that encompasses the weaponization of tech debt. The EU is applying serious pressure on the mobile device giants with [Digital Markets Act](https://en.wikipedia.org/wiki/Digital_Markets_Act). The [fines](https://en.wikipedia.org/wiki/Digital_Markets_Act#Violations_and_fines) the EU is levying are of a different magnitude than the "cost of doing business" fines the big tech companies have been able to swallow painlessly up to now. The situation is not static and there's a future where we'll be able to build what we need where we need it.

## Postscript: Circumstances Alter Cases
For a single developer who wants to solve this problem once and use it for a couple fun projects, the conclusions above hold. For a larger organization with some budget, an approach like xplatter is still sound. If delivering performance-critical, cross-platform mobile applications is key to your business, the 1st party platform requirements churn is a fixed cost. Whether you pay it in manually maintained binding projects per platform or in a binding generator project, it's all the same. That being the case, saving on the overhead and risk of manually maintaining extensive cross-platform bindings is an unqualified win. If you have multiple such products and can reap the benefit over all of them you're coming out well ahead.
