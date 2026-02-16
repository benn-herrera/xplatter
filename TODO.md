# TODO

## examples/hello-xplattergy

* app-web builds and runs properly when IMPL is cpp but not the others. fix each in a separate task.
  * DONE - fix IMPL=c case
    * impl-c build needs to build consumable package for all specified targets the way impl-cpp does
  * fix IMPL=rust case
    * impl-rust build needs to build consumable package for all specified targets the way impl-cpp does
  * fix IMPL=go case
    * impl-go build needs to build consumable package for all specified targets the way impl-cpp does
  * test with ```make test-examples-hello-impl-app-matrix```
