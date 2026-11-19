## Maslow: `pass`
Profile `quick` · spec `testdata/valid/retries.yaml` from `head` · maslow 0.9.0

Nothing failed.

### Flaky results

These failed and then passed on a retry. A flaky check or spec contract counts as a pass; a flaky overlay contract needs review, and a flaky base run is not red.

| Section | Item | Attempts | First failure |
|---|---|---|---|
| check | integration | 2 | exit status 1 |
| contract | api/health | 3 | expected exit code 0, got 1 |
