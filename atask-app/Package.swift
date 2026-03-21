// swift-tools-version: 6.0
import PackageDescription

let package = Package(
    name: "atask",
    platforms: [.macOS(.v15)],
    dependencies: [
        .package(url: "https://github.com/groue/GRDB.swift", from: "7.0.0"),
    ],
    targets: [
        .executableTarget(
            name: "atask",
            dependencies: [
                .product(name: "GRDB", package: "GRDB.swift"),
            ],
            path: "Sources",
            resources: [
                .copy("Resources/Fonts"),
            ]
        ),
        .testTarget(
            name: "ataskTests",
            dependencies: ["atask"],
            path: "Tests"
        ),
    ]
)
