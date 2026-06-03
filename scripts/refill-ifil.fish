#!/usr/bin/env fish
# Delete all iFIL rows at the given heights and refill each tipset.
# Usage:  ./refill-ifil.fish <height1> [height2 ...]
#
# Safe to run alongside live indexer: refill writes go through
# ON CONFLICT DO NOTHING. The DELETE step nukes phantom daemon-era rows
# at those heights; refill then re-inserts only the real iFIL Transfers.

if test (count $argv) -lt 1
    echo "Usage: $0 <height1> [height2 ...]"
    exit 1
end

set -l csv (string join "," $argv)

echo "==> DELETE ifil rows at heights: $csv"
PGPASSWORD=hiro psql -h 127.0.0.1 -p 5433 -U postgres -d mainnet -c "
  SELECT count(*) AS before, height FROM ifil
   WHERE height IN ($csv) GROUP BY height ORDER BY height;
  DELETE FROM ifil WHERE height IN ($csv);
"

echo "==> Refill each tipset"
for h in $argv
    kubectl -n glif exec deploy/idx-deployment-mainnet -- /app/idx refill --from $h --to $h
end

echo "==> Done. Re-check with:  ./invariants pool ifil --per-depositor"
