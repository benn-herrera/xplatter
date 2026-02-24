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

## Code State Review - use architecture review agent
* review the implementation of the xplatter tool (src/)
* these items are particular concerns but are not an exhaustive list
  * duplicate code blocks
  * excessive functionality multiplexing in single functions (death by flags)
    * e.g. functions with excessively moded behavior that would be clearer split up or re-designed
  * abstraction layering violations
  * abstraction inversions (simple capabilities achieved via complex use of insufficient API)
  * overexposure of data - wide propagation of structures needed for narrow purposes
* some of the above may require discussion in the context of pragmatism vs. purity tradeoffs
* overriding principle: tool user experience quality and consistency is the highest priority.
  * truism: users just want good stuff, they don't care about the developer's problems.
  * this tool addresses a messy, hard problem. that's why there's value in providing a solution.
  * some features are best left out if the cost in complexity would compromise the rest of the tool or significantly impede refinement and development.
* produce a list of areas of concern. 
  * rate each of them on a 1-10 scale
    * 1: possible concern, you need more information or guidance
    * 10: absolutely a problem, no question that it needs to be addressed

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
