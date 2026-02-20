# TODO

* to obtain tasking context read ./AGENTS.md and ./ARCHITECTURE.md
* items on the todo list are addressed one encapsulated (## headed) issue at a time in top down order.
* when completed they are marked done (## DONE - [description]) and moved to the end of the file for archival purposes

## separation of the lifecycle functions from the interface definition and methods in the yaml API spec is clunky
* an interface should allow specifying one or more constructor methods that will map to c create functions.
  * this live in a section called 'constructors' that is a sibling to 'methods'
  * constructors must be named create or start with create_
    * e.g. create_from_string
    * name collision is not allowed (overloading not supported)
* if an interface has one or more constructor methods it automatically gets a destructor without need for declaring it manually.
  * regardless of how many constructors are declared there is only ever one destructor. 
* if an interface has no constructor methods it effectively acts as a namespace or empty class with all static methods.

## _IGNORE THIS LINE AND EVERYTHING BELOW IT IN THIS FILE - STAGING AREA FOR FUTURE WORK_

## in GC'd languages (Swift, Kotlin, JavaScrip)   
  * the generated bindings should map constructors (create functions) to setup functions that replace the string 'create' in the name with 'setup' 
  * a destructor should map a bound to a function called 'teardown'. 
    * teardown should clear the cached handle after invoking the destructor function.
    * post-teardown state should be equivalent to pre-setup state (i.e. safe to call setup again after)
  * calling any combination of setup functions twice without having called teardown should raise an exception
  * calling 'teardown' multiple times should be safe
  * method binding wrappers should verify a non-null (zero) handle and raise an exception if verification fails.

  ## generated cpp abstract API includes lifecycle functions as virtual methods.
* this is nonsensical. the generated c shims handle invocation of constructor and destructor.
* in the case of C++ the lifecyle methods already map to constructor and destructor. including empty stubs is pointless and confusing.

## ensure method and constructor names are not allowed to collide. name collisions must be fatal errors.

## error messages for violations of constraints that can't be caught by the validator must produce error message that include file path and line number.

## architecture question: do we want to allow for interface inheritance?
* complexity cost vs. flexibility benefit?
