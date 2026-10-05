# Base64 tails containing form delimiters

A captured emulator `setWanderingGhost.spd` request returned HTTP 500 because unused bytes after valid `replayData` padding contained both `&` and `;`. Splitting the form at ampersands before trimming left an extra fragment that Go's strict query parser rejected.

The transport helper now remembers when it has removed an unused tail from `replayData` or `NPRoomID`. Subsequent unkeyed ampersand fragments remain part of that tail until the next `key=value` field. That next field is retained unchanged, so legitimate `ver`, `pre`, and other fields survive and malformed ordinary values still fail strict parsing.

For example:

```
replayData=QUJDRA==unused&junk;tail&ver=100&pre=2
```

is normalized to:

```
replayData=QUJDRA==&ver=100&pre=2
```

This is intentionally limited to a recognized opaque field with valid Base64 alphabet before literal padding and an actual unused tail after it. Unpadded values, encoded padding, ordinary fields, and malformed data before padding retain their previous behavior. A subsequent keyed fragment always resumes ordinary parsing; there is no global semicolon allowance or suppression of query-parser errors.

Regression tests cover both supported fields, multiple delimiter fragments, preservation of following fields, and rejection of malformed ordinary fields after normalization. Fixtures are synthetic; captured client memory and room tokens are not published.

This fixes an ambient-ghost upload failure. It is not a demonstrated fix for repeated invasions stopping until a host re-enters the game. That separate issue needs host-side matchmaking/session evidence.
