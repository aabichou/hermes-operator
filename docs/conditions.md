# Conditions Catalogue

This document is the authoritative reference for all `status.conditions` emitted by hermes-operator controllers. Condition types are defined as constants in `api/v1/hermesinstance_types.go` so Plans 4–6 and this document stay aligned.

## HermesInstance conditions (Plan 2)

| Type | When True | When False | Reason codes |
|---|---|---|---|
| `Ready` | All subsystems reconciled and STS ready | Any subsystem failed or STS not ready | `AllSubsystemsReady`, `StatefulSetNotReady` |
| `StorageReady` | PVC exists | PVC creation failed | `Reconciled`, `Error` |
| `ConfigReady` | ConfigMap + workspace ConfigMap up to date | One of them failed | `Reconciled`, `Error` |
| `SecretsReady` | Gateway-token Secret exists (placeholder body) | Secret create failed | `Reconciled`, `Error` |
| `NetworkPolicyReady` | NP exists (or correctly absent when disabled) | Failed to (re)create / delete | `Reconciled`, `Error` |
| `RBACReady` | SA + Role + RoleBinding exist | Any of the three failed | `Reconciled`, `Error` |
| `ServiceReady` | Service exists | Service create / update failed | `Reconciled`, `Error` |
| `PDBReady` | PDB exists (or correctly absent) | PDB op failed | `Reconciled`, `Error` |
| `HPAReady` | HPA exists (or correctly absent) | HPA op failed | `Reconciled`, `Error` |
| `IngressReady` | Ingress exists (or correctly absent) | Ingress op failed | `Reconciled`, `Error` |
| `ServiceMonitorReady` | ServiceMonitor exists OR Prometheus-Operator CRDs absent (skipped) | ServiceMonitor op failed | `Reconciled`, `Error` |
| `PrometheusRuleReady` | PrometheusRule exists OR Prometheus-Operator CRDs absent (skipped) | PrometheusRule op failed | `Reconciled`, `Error` |

### `ProfileStoreReady` (Plan 3 refinement)

`ProfileStoreReady` is set by the Honcho step in the reconcile chain. Plan 3 refines its reasons:

| Status | Reason | Meaning |
|---|---|---|
| `True` | `Disabled` | `spec.profileStore.honcho.enabled` is false; the operator did not create Honcho resources. |
| `True` | `Ready` | Honcho Deployment has >=1 ready replica. |
| `False` | `DeploymentNotReady` | Honcho Deployment is missing, scaling up, or its readiness probe is failing. |

## HermesClusterDefaults conditions

| Type | When True | When False | Reason codes |
|---|---|---|---|
| `Ready` | name == "cluster" | otherwise | `Singleton`, `InvalidName` |

## HermesSelfConfig conditions

| Type | Status=True meaning | Reasons |
|---|---|---|
| `Applied` | The SSA writes succeeded for every requested action. | `SSASuccess` |
| `Denied` | Policy or validation rejected the request. No mutation occurred. | `PolicyViolation`, `InstanceNotFound`, `ProtectedPath`, `InvalidPatch`, `SSAConflict` |
| `Pending` | The controller has accepted the SelfConfig but not yet attempted apply. | `Accepted` |

Phase derives from conditions: `Applied → Applied`, `Denied → Denied`, otherwise `Pending`.

## BackupReady

Reflects the state of scheduled backups.

| Status | Reason | When |
|---|---|---|
| True | `Scheduled` | A backup CronJob is configured and the most recent run succeeded. |
| False | `S3CredentialsMissing` | `spec.backup.s3.credentialsSecretRef` does not resolve. |
| False | `PersistenceDisabled` | `spec.storage.persistence.enabled=false` — scheduled backups require persistence. |
| (absent) | — | `spec.backup.schedule` is empty. |

## RestoreApplied

Terminal — once True, immutable for the lifetime of the instance.

| Status | Reason | When |
|---|---|---|
| True | `RestoreCompleted` | `status.restoredFrom == spec.restoreFrom`. |
| False | `Restoring` | `init-restore` init container in progress. |
| False | `RestoreFailed` | `init-restore` exited non-zero. |

## AutoUpdated

The outcome of the most recent auto-update cycle.

| Status | Reason | When |
|---|---|---|
| True | `UpToDate` | The current tag is the highest in the channel. |
| True | `Confirmed` | A rollout completed and passed readiness watch. |
| False | `RolloutInFlight` | A rollout is currently being watched. |
| False | `RolledBack` | The most recent rollout failed; image reverted. |
| False | `NoMatchingTag` | No tag in the registry matches the channel. |
| False | `SuppressedKnownFailure` | The highest matching tag equals `status.autoUpdate.lastFailedTag`. |

## AutoUpdateRolledBack

Present only after a rollback. The reason embeds the failed tag.

| Status | Reason | When |
|---|---|---|
| True | `RolledBackFrom_<tag>` | A rollback completed. The message describes why (deadline elapsed or probeFailureThreshold reached). |

The condition is removed on the next successful `AutoUpdated=True` (reason=Confirmed) cycle.

## MigrationCompleted

Terminal — once True, immutable for the lifetime of the instance.

| Status | Reason | When |
|---|---|---|
| True | `MigrationCompleted` | The `init-migrate-from-openclaw` init container exited 0. |
| False | `MigrationFailed` | The migration init container exited non-zero. |
| (absent) | — | `spec.migration.fromOpenClaw` is unset. |
