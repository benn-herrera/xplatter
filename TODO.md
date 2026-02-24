# TODO

* items on the todo list are addressed one encapsulated (## headed) issue at a time in top down order. The task is to be accomplished starting in planning mode.
  * Discussions tasks stay in planning mode. 
  * Writing tasks (code or otherwise) move on to execution mode.
  * If there are agents available that are well aligned for planning and/or execution, involve them
    * e.g. an architecture discussion should involve the architecture review agent, a tech writing task should use the writing agent, etc.
      * if the architecture in question requires platform specific expertise include the specialist's input.
    * you can use more than one agent per task
    * you can switch agents between planning and execution modes if that makes sense (e.g. hand off from architecture review to web specialist for implementation) 
* unless otherwise instructed in the task item read ./AGENTS.md and ./ARCHITECTURE.md to acquire context if you have not already done so.
* when a task is completed mark it as done (## DONE - [description]) and move it to the end of the file for archival purposes.

## generate a docs/PLATFORM_HAZARDS.md outlining the platform specific hazards that are coped with in the build process for the example impls and platform hello apps. include all of the items following list plus anything else you see in the Makefiles that are platform-specific issues. Particularly things that are more 'lore' than 'documented technology'. This document should prize value density. Explanations must explain the workaround and what it fixed in a concise manner.
* use of zig cc/c++ instead of cl.exe on MS for CGO compiler on Window
  * trying to use cl.exe is a painful dead end
  * zig is a simple dependency (and will be a fully supported system language option soon)
* use of sed to to convert \ to / for paths in Makefiles
  * this may be entirely gnu make specific
  * xplatter has no specifically required build system
  * if you want to use make for cross plat building with windows included, it's important 
* generation of .lib file from a header and .dll via generated .def in impl-go on Windows
  * use of dumpbin from Windows SDK would also work
  * header symbol extraction was simpler for this example
    * any other way of getting the symbol list will work
    * windows sdk lib.exe would work as well as zig dlltool does

## DONE - architecture discussion: is the feature for generating scaffold makefiles worth it
Decision: keep generators. The generated Makefile is correctly parameterized from the API definition (API_NAME, LIB_NAME, BUILD_MACRO, GEN_DIR pre-filled; only targets declared in the API included). Reference examples alone would require manual adaptation. The 1,112 lines of generator code (makefile.go + 4 impl_makefile_*.go) are well-factored with 64% shared infrastructure and no known pain points. GNU Make only — no additional build system generators. Key risk (example Makefiles drifting from generators) is mitigated by the scaffold-once semantics.


## DONE - architecture discussion: leveraging zig to reduce developer and user setup complexity
Decision: windows sdk is how real work is done. replacing with zig is nice for the example project but doesn't help users. Update the way MSVC dependency is managed to better practices.
* can we use its c and c++ compiler drop in replacement capabilities to simplify our Makefiles
  * maybe eliminate use of cmake?
  * maybe eliminate need for cl.exe at all on windows?
* consider the above and we'll begin a discussion


## _IGNORE THIS LINE AND EVERYTHING BELOW IT IN THIS FILE - STAGING AREA FOR FUTURE WORK_
 
## in GC'd languages (Swift, Kotlin, JavaScript)   
  * the generated bindings should map constructors (create functions) to setup functions that replace the string 'create' in the name with 'setup' 
  * a destructor should map a bound to a function called 'teardown'. 
    * teardown should clear the cached handle after invoking the destructor function.
    * post-teardown state should be equivalent to pre-setup state (i.e. safe to call setup again after)
  * calling any combination of setup functions twice without having called teardown should raise an exception
  * calling 'teardown' multiple times should be safe
  * method binding wrappers should verify a non-null (zero) handle and raise an exception if verification fails.

## users are going to have their own preferences for build systems in their projects. the extent to which we generate Makefiles to help them may be too opinionated. investigate just how much of an obstruction the current behavior is toward the user's preferences.

## ensure method and constructor names are not allowed to collide. name collisions must be fatal errors.

## error messages for violations of api definition constraints in the yaml file that can't be caught by the schema validator must produce error message that include file path and line number.

## architecture question: do we want to allow for interface inheritance?
* complexity cost vs. flexibility benefit?

## returning strings only as flatbuffer BoxedString that requires implementer to hold an allocation indefinitely is unacceptably burdensome on xplatter users (API implementers) - think about this problem more carefully.
