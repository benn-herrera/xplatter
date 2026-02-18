/*
 * Swift desktop terminal app that loads the hello_xplatter shared library
 * and exercises the API through the generated Swift binding.
 */

import Foundation

func main() throws {
    print("=== hello_xplatter desktop app (Swift) ===\n")

    let greeter = try Greeter.createGreeter()

    // Discover backing implementation
    if let probe = try? greeter.sayHello(name: ""), let impl = probe.apiImpl {
        print("Backing implementation: \(String(cString: impl))")
    }

    print("Enter a name (or 'exit' to quit): ", terminator: "")
    fflush(stdout)

    while let line = readLine() {
        let trimmed = line.trimmingCharacters(in: .whitespacesAndNewlines)

        if trimmed == "exit" || trimmed == "quit" {
            break
        }

        if trimmed.isEmpty {
            print("Enter a name (or 'exit' to quit): ", terminator: "")
            fflush(stdout)
            continue
        }

        do {
            let result = try greeter.sayHello(name: trimmed)
            print(String(cString: result.message))
        } catch {
            print("say_hello failed: \(error)")
        }

        print("Enter a name (or 'exit' to quit): ", terminator: "")
        fflush(stdout)
    }

    print("Goodbye!")
}

try main()
