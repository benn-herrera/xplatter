# TODO

* to obtain tasking context read ./AGENTS.md and ./ARCHITECTURE.md
* items on the todo list are addressed one encapsulated (## headed) issue at a time in top down order.
* when completed they are marked done (## DONE - [description]) and moved to the end of the file for archival purposes

## packaging example rust impl fails
* ```make package-hello-rust``` yields error:
   ```error: couldn't read `src/../generated/hello_xplatter_types.rs`: No such file or directory (os error 2)
 --> src/lib.rs:5:1```

## impl-cpp and impl-rust are incorrectly using generated files
* they are copying files out of generated/ for use in their build processes instead of consuming them in place.
* this results in generated files as siblings to authored files, which is inconsistent with our project pattern.


## example app-android builds but is crashing on launch.
* error is: ```java.lang.UnsatisfiedLinkError: dlopen failed: library "libc++_shared.so" not found: needed by /data/app/~~NNPSSFRBY4q2vhgS_S8DUg==/com.example.helloapp-RBRAwfVn6s8wJx39c7gnOg==/lib/arm64/libhello_xplatter.so in namespace clns-7```

## impl-go is implementing twice - once for WASM and once for everyone else. there needs to be a way for the user of xplatter to implement once.
