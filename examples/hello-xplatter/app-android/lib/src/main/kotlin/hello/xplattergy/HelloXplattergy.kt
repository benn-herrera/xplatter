package hello.xplattergy

class HelloErrorCodeException(val errorCode: Int) : Exception("Error code: $errorCode")

data class HelloGreeting(val message: String?, val apiImpl: String?)

class Greeter internal constructor(internal val handle: Long) : AutoCloseable {
    fun sayHello(name: String): HelloGreeting {
        return HelloXplattergy.nativeGreeterSayHello(handle, name)
    }

    override fun close() {
        HelloXplattergy.nativeLifecycleDestroyGreeter(handle)
    }
}

object HelloXplattergy {
    init {
        System.loadLibrary("hello_xplattergy")
    }

    fun createGreeter(): Greeter {
        val result = nativeLifecycleCreateGreeter()
        if (result[0] != 0L) throw HelloErrorCodeException(result[0].toInt())
        return Greeter(result[1])
    }

    external fun nativeLifecycleCreateGreeter(): LongArray
    external fun nativeLifecycleDestroyGreeter(greeter: Long): Unit
    external fun nativeGreeterSayHello(greeter: Long, name: String): HelloGreeting
}
