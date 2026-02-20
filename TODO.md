# TODO

* to obtain tasking context read ./AGENTS.md and ./ARCHITECTURE.md
* items on the todo list are addressed one encapsulated (## headed) issue at a time in top down order.
* when completed they are marked done (## DONE - [description]) and moved to the end of the file for archival purposes

## architectural question
* does it make sense to have so many multiline text blocks interspersed in the generator source files? what about
  * use complete file templates with a substitution syntax for the variant text elements. 
    * subsitution syntax would be something like ${{PARAM_NAME}}
    * the locations consuming the template must have a complete mapping of the necessary parameters to fill out the template. unfulfilled substitutions are a fatal error. 
    * if there's already a clean, standard syntax for this let's use it.
  * complete file templates live as multiline strings in the source files that reference them.
    * does go allow forward declarations of file level constants?
    * would be nice to have these templates live at the bottoms of the implementation files instead of the tops.

## impl-go is implementing twice - once for WASM and once for everyone else. there needs to be a way for the user of xplatter to implement once.
