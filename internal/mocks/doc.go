// Package mocks contains mockgen-generated test doubles for the interfaces
// consumed by unit tests across the project.
//
// Regeneration policy:
//   - go:generate directives live next to the corresponding interface
//     declarations in their source packages (cache, fetcher, services, sysproxy),
//     so changing an interface uncovers its mock in place.
//   - Regenerate everything with: go generate ./...
//   - Requires: go install go.uber.org/mock/mockgen@latest
//   - Only interfaces that already have a consuming test are mocked.
//     Add a new directive together with the test that consumes it.
package mocks
