import SwiftUI
import HelloXplattergyBinding

struct ContentView: View {
    @State private var nameInput = "xplattergy"
    @State private var greetingOutput = ""
    @State private var greeter: Greeter?

    var body: some View {
        VStack(spacing: 20) {
            TextField("Enter a name", text: $nameInput)
                .textFieldStyle(.roundedBorder)
            Button("Greet") { greet() }
                .buttonStyle(.borderedProminent)
            Text(greetingOutput)
        }
        .padding()
        .onAppear {
            greeter = try? Greeter.createGreeter()
        }
    }

    private func greet() {
        guard let g = greeter,
              let r = try? g.sayHello(name: nameInput) else { return }
        greetingOutput = String(cString: r.message)
    }
}
