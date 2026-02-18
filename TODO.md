# TODO

* to obtain tasking context read ./AGENTS.md and ./ARCHITECTURE.md
* items on the todo list are addressed one encapsulated (## headed) issue at a time in top down order.
* when completed they are marked done (## DONE - [description]) and moved to the end of the file for archival purposes


## impl-go is implementing twice - once for WASM and once for everyone else. see if there's a workaround.

## example app-android project not debuggable from Android Studio like iOS xcode project is. find out why.

## examine how the impl dependencies are managed. 
* it looks like the consumers are still doing to much work.
* the impls should not just produce all the code files or compile a simple library. they should be producing *packages* that are suitable to passing off to another project as part of an sdk.
