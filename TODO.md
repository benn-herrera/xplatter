# TODO

* to obtain tasking context read ./AGENTS.md and ./ARCHITECTURE.md
* items on the todo list are addressed one encapsulated (## headed) issue at a time in top down order.
* when completed they are marked done (## DONE - [description]) and moved to the end of the file for archival purposes

## generated cpp abstract API includes lifecycle functions as virtual methods.
* this is nonsensical. the generated c shims handle invocation of constructor and destructor.
* in the case of C++ the lifecyle methods already map to constructor and destructor. including empty stubs is pointless and confusing.

## _IGNORE THIS LINE AND EVERYTHING BELOW IT IN THIS FILE - STAGING AREA FOR FUTURE WORK_

## separation of the lifecycle functions from the interface definition and methods in the yaml API spec is clunky
* an interface should allow specifying one or more constructor methods that will map to c create functions.
  * this live in a section called 'constructors' that is a sibling to 'methods'
  * constructors must be named create or start with create_
    * e.g. create_from_string
    * name collision is not allowed 
* if an interface has one or more constructor methods it automatically gets a destructor without need for declaring it manually.
* if an interface has no constructor methods it effectively acts as a namespace or empty class with all static methods.

## ensure method and constructor names are not allowed to collide. name collisions must be fatal errors.

## error messages for violations of constraints that can't be caught by the validator must produce error message that include file path and line number.
