echo '#!/bin/bash
while true; do echo "UID PID %usr %system %guest %wait %CPU CPU Command"; sleep 1; done' > pidstat
chmod +x pidstat