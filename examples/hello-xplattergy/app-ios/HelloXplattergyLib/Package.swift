// swift-tools-version: 5.9
import PackageDescription

let package = Package(
    name: "HelloXplattergyLib",
    platforms: [.iOS(.v15)],
    products: [
        .library(name: "HelloXplattergyLib", targets: ["HelloXplattergyBinding"]),
    ],
    targets: [
        .binaryTarget(name: "CHelloXplattergy", path: "../build/HelloXplattergy.xcframework"),
        .target(
            name: "HelloXplattergyBinding",
            dependencies: ["CHelloXplattergy"],
            path: "Sources/HelloXplattergyBinding",
            linkerSettings: [.linkedLibrary("c++")]
        ),
    ]
)
