// Package service contains the application (use case) layer. Services
// orchestrate domain logic and infrastructure through ports —
// interfaces defined in this package and implemented by adapters in
// internal/store and internal/gateway. This layer knows nothing about
// HTTP, Gin, SQL drivers, or external SDKs.
package service
