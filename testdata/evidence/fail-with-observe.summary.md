## Maslow: `fail`
Profile `quick` · spec `testdata/valid/minimal.yaml` from `head` · maslow 0.1.0

| Section | Item | Status | Detail |
|---|---|---|---|
| check | build | fail | exit status 1 |

### Observed policy rules (not enforced)

These rules run in observe mode. Their findings do not change the verdict.

| Kind | Rule | Would be | Detail |
|---|---|---|---|
| gated | maslow.yaml | would_need_review | gated path changed; review by @owner |
