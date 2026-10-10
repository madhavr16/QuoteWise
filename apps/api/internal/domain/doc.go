// Package domain holds QuoteWise's pure business core: entities,
// value objects, and invariants (money as decimals, quotation review
// rules, comparison warnings).
//
// Dependency rule: domain imports nothing from other layers — no
// HTTP, no SQL, no SDKs. Every other layer may import domain.
//
// Milestone 1 has no business entities yet; procurement entities and
// the comparison engine land in Milestones 2 and 4 (see
// docs/implementation-plan.md).
package domain
