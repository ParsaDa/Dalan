# Dalan 0.7.3 — release notes

Fixes for the in-panel update and for Compare methods.

## Fixed

- **The panel updated but still offered the same update.** The 0.7.1 and 0.7.2 binaries reported their version as 0.7.0, because the version file was not raised before the tag. The update check then saw a newer release every time, even right after an update. A release build now writes its own tag into the version file, so a binary always reports the version it was released as. If you are on 0.7.1 or 0.7.2, update once more; from 0.7.3 on the panel shows the right version.
- **Compare methods failed every method with `i/o timeout`.** The message now says whether the exit server hung up or sent no reply within 5 seconds, and that the exit server needs a Dalan version with the link speed test. After three methods in a row get no answer the run stops and says why.

## Changed

- Compare methods no longer limits a run to 60 methods.
