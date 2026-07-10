# identity (planned, Go)

Multi-tenant identity and access. Resolves the virtual API keys the gateway sees into
`(tenant, customer, feature)` and guards the control-plane APIs.

- tenants, projects, virtual keys, RBAC
- key → caller resolution used by the gateway inbound adapter
- SSO/SCIM in the enterprise tier

Status: the gateway currently reads caller headers with demo defaults; this service
replaces that in Phase 2.
