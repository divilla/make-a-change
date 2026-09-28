# Final 016 coverage inventory and uncovered blocks

Fresh independent profiles from `backend/.coverage/unit` and `backend/.coverage/api`.
All 28 production Go files in all ten packages are inventoried below; domain has
no executable statements. Tooling and legacy HTTP code are excluded by contract.
No source was excluded and no profiles were merged. Counts use Go blocks, not
averages or rounded percentages. Production source is unchanged from merged R6
`62d9f4f5ff2fd55629b913b50e06932397902585`.

| Package | Production files | Unit | APIHydra |
| --- | --- | --- | --- |
| cmd/server | lifecycle.go, main.go, router.go | 77/91 | 76/91 |
| internal/change | api.go, change_view.go, repo.go, service.go | 389/389 | 376/389 |
| internal/domain | change.go, dtos.go, epic.go, health.go, project.go, test_case.go | 0/0 | 0/0 |
| internal/epic | api.go, repo.go, service.go | 114/114 | 110/114 |
| internal/error | errors.go | 48/48 | 39/48 |
| internal/health | api.go, repo.go, service.go | 21/21 | 21/21 |
| internal/project | api.go, repo.go, service.go | 116/116 | 112/116 |
| internal/testcase | api.go, repo.go, service.go | 105/105 | 100/105 |
| pkg/config | config.go | 31/31 | 22/31 |
| pkg/markdown | markdown.go | 10/10 | 8/10 |
| **Total** | **28 files** | **911/925 (98.4865%)** | **864/925 (93.4054%)** |

## Exact uncovered source blocks

Ranges are Go coverage positions (line.column), with the actual statement count.
All non-main gaps have meaningful existing unit failure/boundary tests. HTTP ID
validation deliberately prevents re-executing equivalent direct-service guards.
The healthy server campaign cannot cover fatal startup and still be a complete
one-server campaign; 14 main orchestration statements remain outside unit execution.
No missing required scenario or numerical target remains. A future separately
specified startup-process campaign could test process failures; it must keep its
own evidence and must not be mixed into this APIHydra profile.

### Unit

| Source and function | Block | Statements | Behavior and practical obstacle |
| --- | --- | --- | --- |
| [cmd/server/main.go](../cmd/server/main.go:15) `main` | `15.13,21.21` | 5 | Main flags, signal lifecycle and fatal exit are process orchestration; lifecycle/start/config have direct unit tests. |
| [cmd/server/main.go](../cmd/server/main.go:21) `main` | `21.21,23.3` | 1 | Main flags, signal lifecycle and fatal exit are process orchestration; lifecycle/start/config have direct unit tests. |
| [cmd/server/main.go](../cmd/server/main.go:24) `main` | `24.19,26.3` | 1 | Main flags, signal lifecycle and fatal exit are process orchestration; lifecycle/start/config have direct unit tests. |
| [cmd/server/main.go](../cmd/server/main.go:24) `main` | `24.2,24.19` | 1 | Main flags, signal lifecycle and fatal exit are process orchestration; lifecycle/start/config have direct unit tests. |
| [cmd/server/main.go](../cmd/server/main.go:28) `main` | `28.2,30.49` | 3 | Main flags, signal lifecycle and fatal exit are process orchestration; lifecycle/start/config have direct unit tests. |
| [cmd/server/main.go](../cmd/server/main.go:30) `main` | `30.49,30.75` | 1 | Main flags, signal lifecycle and fatal exit are process orchestration; lifecycle/start/config have direct unit tests. |
| [cmd/server/main.go](../cmd/server/main.go:30) `main` | `30.89,33.3` | 2 | Main flags, signal lifecycle and fatal exit are process orchestration; lifecycle/start/config have direct unit tests. |

### APIHydra

| Source and function | Block | Statements | Behavior and practical obstacle |
| --- | --- | --- | --- |
| [cmd/server/lifecycle.go](../cmd/server/lifecycle.go:26) `run` | `26.16,28.3` | 1 | Fatal startup, parse/listen/CORS failure, Serve termination or failed shutdown; incompatible with a healthy owned lifecycle. |
| [cmd/server/lifecycle.go](../cmd/server/lifecycle.go:35) `run` | `35.27,36.42` | 1 | Fatal startup, parse/listen/CORS failure, Serve termination or failed shutdown; incompatible with a healthy owned lifecycle. |
| [cmd/server/lifecycle.go](../cmd/server/lifecycle.go:43) `run` | `43.24,47.3` | 3 | Fatal startup, parse/listen/CORS failure, Serve termination or failed shutdown; incompatible with a healthy owned lifecycle. |
| [cmd/server/lifecycle.go](../cmd/server/lifecycle.go:54) `start` | `54.16,56.3` | 1 | Fatal startup, parse/listen/CORS failure, Serve termination or failed shutdown; incompatible with a healthy owned lifecycle. |
| [cmd/server/lifecycle.go](../cmd/server/lifecycle.go:58) `start` | `58.16,61.3` | 2 | Fatal startup, parse/listen/CORS failure, Serve termination or failed shutdown; incompatible with a healthy owned lifecycle. |
| [cmd/server/lifecycle.go](../cmd/server/lifecycle.go:66) `start` | `66.16,70.3` | 3 | Fatal startup, parse/listen/CORS failure, Serve termination or failed shutdown; incompatible with a healthy owned lifecycle. |
| [cmd/server/main.go](../cmd/server/main.go:30) `main` | `30.89,33.3` | 2 | Fatal startup, parse/listen/CORS failure, Serve termination or failed shutdown; incompatible with a healthy owned lifecycle. |
| [cmd/server/router.go](../cmd/server/router.go:46) `newRouter` | `46.16,48.3` | 1 | Fatal startup, parse/listen/CORS failure, Serve termination or failed shutdown; incompatible with a healthy owned lifecycle. |
| [cmd/server/router.go](../cmd/server/router.go:89) `jsonErrorHandler` | `89.80,91.3` | 1 | Writing an HTTP error itself fails; a valid client response cannot demonstrate a broken response writer. |
| [internal/change/repo.go](../internal/change/repo.go:102) `Documents` | `102.16,104.3` | 1 | Query fails after successful parent preflight; requires timing a second query failure, not a whole-DB outage. |
| [internal/change/repo.go](../internal/change/repo.go:109) `Documents` | `109.88,111.4` | 1 | Scan/type-conversion failure with correct schema and valid stored data; covered by unit row fakes. |
| [internal/change/repo.go](../internal/change/repo.go:131) `Artifacts` | `131.57,133.4` | 1 | Scan/type-conversion failure with correct schema and valid stored data; covered by unit row fakes. |
| [internal/change/repo.go](../internal/change/repo.go:185) `UpdateChangeTypes` | `185.16,187.3` | 1 | Exec failure/missing row after successful config preflight; requires concurrent change or timed failure. |
| [internal/change/repo.go](../internal/change/repo.go:188) `UpdateChangeTypes` | `188.29,190.3` | 1 | Exec failure/missing row after successful config preflight; requires concurrent change or timed failure. |
| [internal/change/repo.go](../internal/change/repo.go:60) `List` | `60.195,62.4` | 1 | Scan/type-conversion failure with correct schema and valid stored data; covered by unit row fakes. |
| [internal/change/service.go](../internal/change/service.go:121) `CreateChange` | `121.17,123.4` | 1 | UUID entropy failure; valid API input cannot force operating-system entropy failure. |
| [internal/change/service.go](../internal/change/service.go:139) `UpdateChangeTypes` | `139.17,141.3` | 1 | Invalid ID direct-service guard; HTTP validation rejects before this layer. |
| [internal/change/service.go](../internal/change/service.go:253) `DeleteChange` | `253.17,255.3` | 1 | Invalid ID direct-service guard; HTTP validation rejects before this layer. |
| [internal/change/service.go](../internal/change/service.go:34) `ListChanges` | `34.24,36.3` | 1 | Invalid ID direct-service guard; HTTP validation rejects before this layer. |
| [internal/change/service.go](../internal/change/service.go:49) `GetChange` | `49.17,51.3` | 1 | Invalid ID direct-service guard; HTTP validation rejects before this layer. |
| [internal/change/service.go](../internal/change/service.go:89) `Documents` | `89.17,91.3` | 1 | Invalid ID direct-service guard; HTTP validation rejects before this layer. |
| [internal/change/service.go](../internal/change/service.go:96) `Documents` | `96.16,98.3` | 1 | Document query fails after successful existence preflight; deterministic whole-DB outage fails earlier. |
| [internal/epic/repo.go](../internal/epic/repo.go:40) `List` | `40.17,42.4` | 1 | Scan/type-conversion failure with correct schema and valid stored data; covered by unit row fakes. |
| [internal/epic/service.go](../internal/epic/service.go:33) `ListEpics` | `33.24,35.3` | 1 | Invalid ID direct-service guard; HTTP validation rejects before this layer. |
| [internal/epic/service.go](../internal/epic/service.go:48) `GetEpic` | `48.17,50.3` | 1 | Invalid ID direct-service guard; HTTP validation rejects before this layer. |
| [internal/epic/service.go](../internal/epic/service.go:78) `DeleteEpic` | `78.17,80.3` | 1 | Invalid ID direct-service guard; HTTP validation rejects before this layer. |
| [internal/error/errors.go](../internal/error/errors.go:120) `HTTP` | `120.16,122.3` | 1 | Nil/empty helper input or already-normalized HTTP error; normal request call sites do not pass these combinations. |
| [internal/error/errors.go](../internal/error/errors.go:125) `HTTP` | `125.69,127.3` | 1 | Nil/empty helper input or already-normalized HTTP error; normal request call sites do not pass these combinations. |
| [internal/error/errors.go](../internal/error/errors.go:136) `ServerShutdown` | `136.2,136.27` | 1 | Abnormal Serve error; complete campaign requires normal graceful shutdown. |
| [internal/error/errors.go](../internal/error/errors.go:147) `ChangeCreate` | `147.3,147.95` | 1 | Parent disappears after successful config preflight before procedure; concurrent DB race, not deterministic outage. |
| [internal/error/errors.go](../internal/error/errors.go:147) `ChangeCreate` | `147.95,149.4` | 1 | Parent disappears after successful config preflight before procedure; concurrent DB race, not deterministic outage. |
| [internal/error/errors.go](../internal/error/errors.go:34) `Wrap` | `34.33,36.3` | 1 | Nil/empty helper input or already-normalized HTTP error; normal request call sites do not pass these combinations. |
| [internal/error/errors.go](../internal/error/errors.go:63) `Validation` | `63.18,65.3` | 1 | Nil/empty helper input or already-normalized HTTP error; normal request call sites do not pass these combinations. |
| [internal/error/errors.go](../internal/error/errors.go:66) `Validation` | `66.21,68.3` | 1 | Nil/empty helper input or already-normalized HTTP error; normal request call sites do not pass these combinations. |
| [internal/error/errors.go](../internal/error/errors.go:80) `Interpret` | `80.16,82.3` | 1 | Nil/empty helper input or already-normalized HTTP error; normal request call sites do not pass these combinations. |
| [internal/project/repo.go](../internal/project/repo.go:51) `List` | `51.17,53.4` | 1 | Scan/type-conversion failure with correct schema and valid stored data; covered by unit row fakes. |
| [internal/project/service.go](../internal/project/service.go:27) `GetProject` | `27.17,29.3` | 1 | Invalid ID direct-service guard; HTTP validation rejects before this layer. |
| [internal/project/service.go](../internal/project/service.go:53) `DeleteProject` | `53.17,55.3` | 1 | Invalid ID direct-service guard; HTTP validation rejects before this layer. |
| [internal/project/service.go](../internal/project/service.go:61) `Config` | `61.17,63.3` | 1 | Invalid ID direct-service guard; HTTP validation rejects before this layer. |
| [internal/testcase/repo.go](../internal/testcase/repo.go:35) `List` | `35.16,37.3` | 1 | Query fails after successful parent preflight; requires timing a second query failure, not a whole-DB outage. |
| [internal/testcase/repo.go](../internal/testcase/repo.go:42) `List` | `42.108,44.4` | 1 | Scan/type-conversion failure with correct schema and valid stored data; covered by unit row fakes. |
| [internal/testcase/service.go](../internal/testcase/service.go:33) `ListTestCases` | `33.23,35.3` | 1 | Invalid ID direct-service guard; HTTP validation rejects before this layer. |
| [internal/testcase/service.go](../internal/testcase/service.go:59) `UpdateTestCaseDone` | `59.17,61.3` | 1 | Invalid ID direct-service guard; HTTP validation rejects before this layer. |
| [internal/testcase/service.go](../internal/testcase/service.go:67) `DeleteTestCase` | `67.17,69.3` | 1 | Invalid ID direct-service guard; HTTP validation rejects before this layer. |
| [pkg/config/config.go](../pkg/config/config.go:26) `New` | `26.60,27.45` | 1 | Config load/decode panic, unused defaults/environment overrides or colon-prefixed port; one fixed startup configuration. |
| [pkg/config/config.go](../pkg/config/config.go:30) `New` | `30.44,31.45` | 1 | Config load/decode panic, unused defaults/environment overrides or colon-prefixed port; one fixed startup configuration. |
| [pkg/config/config.go](../pkg/config/config.go:40) `applyDefaults` | `40.30,42.3` | 1 | Config load/decode panic, unused defaults/environment overrides or colon-prefixed port; one fixed startup configuration. |
| [pkg/config/config.go](../pkg/config/config.go:43) `applyDefaults` | `43.18,45.3` | 1 | Config load/decode panic, unused defaults/environment overrides or colon-prefixed port; one fixed startup configuration. |
| [pkg/config/config.go](../pkg/config/config.go:46) `applyDefaults` | `46.25,48.3` | 1 | Config load/decode panic, unused defaults/environment overrides or colon-prefixed port; one fixed startup configuration. |
| [pkg/config/config.go](../pkg/config/config.go:52) `applyEnv` | `52.53,54.3` | 1 | Config load/decode panic, unused defaults/environment overrides or colon-prefixed port; one fixed startup configuration. |
| [pkg/config/config.go](../pkg/config/config.go:55) `applyEnv` | `55.45,57.3` | 1 | Config load/decode panic, unused defaults/environment overrides or colon-prefixed port; one fixed startup configuration. |
| [pkg/config/config.go](../pkg/config/config.go:58) `applyEnv` | `58.53,60.3` | 1 | Config load/decode panic, unused defaults/environment overrides or colon-prefixed port; one fixed startup configuration. |
| [pkg/config/config.go](../pkg/config/config.go:65) `Addr` | `65.36,67.3` | 1 | Config load/decode panic, unused defaults/environment overrides or colon-prefixed port; one fixed startup configuration. |
| [pkg/markdown/markdown.go](../pkg/markdown/markdown.go:44) `Parse` | `44.63,47.3` | 2 | Goldmark converter/write failure with in-memory buffer; no supported request deterministically produces it. |
