#!/bin/sh
# Redaction guard (SEC-14). Exits 1 and lists the lines when a tracked file
# holds what the redaction sweeps promised to keep out:
#
#   a. the durable dogfood account address, anywhere;
#   b. the production host in a live surface: server Go sources, docs/usage/,
#      docs/architecture/, README.md, client/README.md;
#   c. any other e-mail address in a tracked text file (rv4-PUB-01). Allowed:
#      a noreply local part; the reserved documentation and test names
#      (example.*, *.example, *.test, *.invalid, *.localhost, *.local);
#      resend.dev, the mail provider's sandbox; x.com, x.co and b.co, the
#      fixture domains of the Go auth tests; and image names such as
#      icon@2x.png. Write <admin-email> or <owner-email> in docs. The report
#      names file:line only, so the log never echoes the address.
#
# (b) does not look at the one shipped default base URL and its test
# (server/internal/apiclient/default_base*.go), the CI workflows, or the dated
# records under docs/changelogs/ and docs/plans/, which keep the host as
# history. The host is read from that shipped default, so neither it nor the
# dogfood address is spelled out in this file and the guard cannot match itself.
#
# Usage: scripts/check-redaction.sh
#   CI runs it as a blocking step; config/git-hooks/pre-commit runs it advisory.
# Exit: 0 clean, 1 findings, 2 cannot tell (no git checkout, no default URL).
set -u

root=$(git rev-parse --show-toplevel 2>/dev/null) || exit 2
cd "$root" || exit 2
base=server/internal/apiclient/default_base.go
host=$(sed -n 's#^[a-z]* DefaultBase = "https\{0,1\}://\([^/"]*\).*#\1#p' "$base" 2>/dev/null)
if [ -z "$host" ]; then
  echo "check-redaction: cannot read the hosted host from $base" >&2
  exit 2
fi

status=0
report() { # report <what> <lines>
  [ -n "$2" ] || return 0
  printf 'redaction: %s\n%s\n' "$1" "$2" >&2
  status=1
}

report "the dogfood account address is in a tracked file; write <dogfood-account>" \
  "$(git grep -n -I -F -e "dogfood@$host" --)"
report "the production host is in a live surface; write <prod-domain>" \
  "$(git grep -n -I -F -e "$host" -- 'server/*.go' \
    ':!server/internal/apiclient/default_base.go' ':!server/internal/apiclient/default_base_test.go' \
    docs/usage docs/architecture README.md client/README.md)"
# (c): git grep -o prints path:line:match, so each allow rule below is a grep
# -v pattern anchored on the match itself; sed keeps only path:line.
report "an e-mail address that is not a placeholder is in a tracked file (value not printed); write <admin-email> or <owner-email>" \
  "$(git grep -n -I -o -i -E -e '[A-Za-z0-9._%+-]+@[A-Za-z0-9-]+(\.[A-Za-z0-9-]+)*\.[A-Za-z]{2,}' -- |
    grep -v -i -E \
      -e ':[0-9]+:noreply@' \
      -e '@([a-z0-9-]+\.)*example\.[a-z]+$' \
      -e '@[^@]*\.(example|test|invalid|localhost|local)$' \
      -e '@([a-z0-9-]+\.)*resend\.dev$' \
      -e '@(x\.com|x\.co|b\.co)$' \
      -e '@[a-z0-9-]+\.(png|jpe?g|gif|webp|svg|ico)$' |
    sed -E 's/^([^:]+:[0-9]+):.*/\1/' | uniq)"
exit "$status"
