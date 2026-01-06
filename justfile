# Helper recipes for working with the SumUp Java SDK repo.

# Show all available commands.
default:
	@just --list

# Regenerate the OpenAPI output using the Go-based wrapper.
generate:
	go -C codegen run . generate --spec ../openapi.json

# Run unit tests for the Go generator code.
go-test:
	go -C codegen test ./...

# Build the SDK (requires JDK 21 available via JAVA_HOME or java_home on macOS).
build:
	JAVA_HOME="${JAVA_HOME:-$(/usr/libexec/java_home -v 21)}" ./gradlew build

# Run the basic example (uses the same Java toolchain helper as build).
example-basic:
	JAVA_HOME="${JAVA_HOME:-$(/usr/libexec/java_home -v 21)}" ./gradlew :examples:basic:run

# Format all Java sources with Spotless.
format:
	JAVA_HOME="${JAVA_HOME:-$(/usr/libexec/java_home -v 21)}" ./gradlew spotlessApply

# Check formatting without modifying files.
format-check:
	JAVA_HOME="${JAVA_HOME:-$(/usr/libexec/java_home -v 21)}" ./gradlew spotlessCheck

# Run the full Gradle test suite.
test:
	JAVA_HOME="${JAVA_HOME:-$(/usr/libexec/java_home -v 21)}" ./gradlew test
