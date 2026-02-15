the examples directory in the distro needs a top level Makefile with targets that build all the available target apps.

move the utility example build and test targets to that file from the main Makefile

add a test-dist Makefile target that manually runs the example make targets in the context of the built distro to verify the distro is set up properly.

do complete dive over everything and make sure it's not being inconceivable.
