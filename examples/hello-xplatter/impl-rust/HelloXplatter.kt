package hello.xplatter

class HelloErrorCodeException(val errorCode: Int) : Exception("Error code: $errorCode")

data class HelloGreeting(val message: String?, val apiImpl: String?)

class Greeter internal constructor(internal val handle: Long) : AutoCloseable {
    fun sayHello(name: String): HelloGreeting {
        return HelloXplatter.nativeGreeterSayHello(handle, name)
    }

    override fun close() {
        HelloXplatter.nativeLifecycleDestroyGreeter(handle)
    }
}

object HelloXplatter {
    init {
        System.loadLibrary("hello_xplatter")
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
