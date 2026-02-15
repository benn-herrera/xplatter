The examples all currently rely on ../../../bin/xplattergy as the xplattergy tool executable.
This works for a developer of xplattergy but not for a consumer of this package because in the distro the binaries are named in the pattern ../../../bin/xplattergy-OS-ARCH

add an executable shell script xplattergy.sh that lives as the top level of the repo.
it checks first for bin/xplattergy (which will be a dev build or built via a call to build_codegen.sh by a xplattergy user). if there is no bin/xplattergy it should determine the OS & ARCH for the local host and construct the correct name for the executable and verify that it exists. on windows don't forget to append .exe to the executable file name.

when an appropriate executable is verified to exist it should be invoked via exec, passing through all command line arguments via "${@}"

if no xplattergy binary exists error out with a message indicating that it needs to be built.

once this is done all of the example makefile targets can be updated to use ../../../xplattergy.sh as the tool executable path. this will make developer and consumer behavior identical.
