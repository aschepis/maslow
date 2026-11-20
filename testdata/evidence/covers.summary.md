## Maslow: `fail`
Profile `quick` · spec `testdata/valid/covers.yaml` from `head` · maslow 0.9.0

| Section | Item | Status | Detail |
|---|---|---|---|
| contract | acceptance/fixed | fail | expected exit code 0, got 1 |

### Requirements

The requirements the run's checks and contracts cover, by their `covers` lists. A requirement fails if any result covering it fails.

| Requirement | Status | Covered by |
|---|---|---|
| REQ-1 | pass | check:unit |
| acme/app#7 | fail | contract:acceptance/fixed, contract:login/rejects-bad-password |
| acme/app#7/ac-2 | pass | contract:login/rejects-bad-password |
