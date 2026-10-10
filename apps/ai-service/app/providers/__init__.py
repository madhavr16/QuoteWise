"""Provider (adapter) layer: LLM and embedding providers.

The deterministic, network-free MockProvider (ADR-009) lands here in
Milestone 3 along with the provider-neutral interfaces. Real provider
adapters are added only after an explicit approval (open decision P2);
no credentials are ever required for development.
"""
