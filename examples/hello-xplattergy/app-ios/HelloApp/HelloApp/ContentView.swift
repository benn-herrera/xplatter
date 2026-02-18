import SwiftUI
import HelloXplattergyBinding

struct ContentView: View {
    @State private var nameInput = "xplattergy"
    @State private var greetingOutput = ""
    @State private var backingImpl = ""
    @State private var greeter: Greeter?

    var body: some View {
        VStack(spacing: 20) {
            if !backingImpl.isEmpty {
                Text(backingImpl)
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }
            TextField("Enter a name", text: $nameInput)
                .textFieldStyle(.roundedBorder)
            Button("Greet") { greet() }
                .buttonStyle(.borderedProminent)
            Text(greetingOutput)
        }
        .padding()
        .onAppear {
            greeter = try? Greeter.createGreeter()
            if let g = greeter,
               let probe = try? g.sayHello(name: ""),
               let impl = probe.apiImpl {
                backingImpl = "Backing implementation: \(String(cString: impl))"
            }
        }
    }

    private func greet() {
        guard let g = greeter,
              let r = try? g.sayHello(name: nameInput) else { return }
        greetingOutput = String(cString: r.message)
    }
}
