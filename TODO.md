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

## ensure method and constructor names are not allowed to collide. name collisions must be fatal errors.

## _IGNORE THIS LINE AND EVERYTHING BELOW IT IN THIS FILE - STAGING AREA FOR FUTURE WORK_
 
## in GC'd languages (Swift, Kotlin, JavaScript)   
  * the generated bindings should map constructors (create functions) to setup functions that replace the string 'create' in the name with 'setup' 
  * a destructor should map a bound to a function called 'teardown'. 
    * teardown should clear the cached handle after invoking the destructor function.
    * post-teardown state should be equivalent to pre-setup state (i.e. safe to call setup again after)
  * calling any combination of setup functions twice without having called teardown should raise an exception
  * calling 'teardown' multiple times should be safe
  * method binding wrappers should verify a non-null (zero) handle and raise an exception if verification fails.

## returning strings only as flatbuffer BoxedString that requires implementer to hold an allocation indefinitely is unacceptably burdensome on xplatter users (API implementers) - think about this problem more carefully.
