#!/usr/bin/env python3
"""rv4-FR-F12: the privacy notice and the CSP must tell the same story.

PRIVACY.md once told readers that the web client sends their IP address to
three Google-operated font and renderer hosts on every page load, while the
CSP in client/nginx.conf allowed 'self' only. The notice has since been
repaired by hand; this guard stops the two drifting apart again. It fails when

  1. PRIVACY.md names a host that the CSP does not allow and that is not in
     SERVER_SIDE (hosts only the server, or a reader who clicks a link, reaches);
  2. the CSP allows a host PRIVACY.md does not name, or allows every host of a
     scheme ('https:' or '*'), which a notice cannot list;
  3. the CSP allows 'self' alone and PRIVACY.md still describes a Google font
     or renderer host, lists a font or renderer as a recipient, or no longer
     says the fonts and the CanvasKit renderer come from the deployment's own
     origin.

Every run also proves, on the live files, that each of those regressions still
turns the guard red (selfproof) and that a notice naming the host a CSP allows
stays green, so the guard follows the CSP instead of banning a fixed list.

Ceiling: hosts are found by a TLD list and by the words in GOOGLE and
CLIENT_LOAD, so a claim written with none of them passes, and the processor
list is not compared with the server's environment variables.

Stdlib only.
Usage: python3 -I scripts/tests/check_privacy_csp_rv4.py [PRIVACY.md [nginx.conf]]
"""

import pathlib
import re
import sys

ROOT = pathlib.Path(__file__).resolve().parents[2]

# Hosts PRIVACY.md may name without the CSP allowing them: the server's GitHub
# recipient and the repository links. A new host in the notice fails until the
# CSP allows it or it is added here, which puts the choice in review.
SERVER_SIDE = {"api.github.com", "github.com"}

# ponytail: a TLD list keeps file names (nginx.conf, environment.md, dev.sh) out
# of the host set; add a TLD when a notice host needs one. An e-mail domain
# (after "@") is not a host the browser loads from.
HOST = re.compile(
    r"(?<![\w.@-])(?:[a-z0-9-]+\.)+"
    r"(?:com|net|org|io|dev|app|ai|cloud|cn|co|xyz|info)(?![\w-])",
    re.I,
)
CSP = re.compile(r'Content-Security-Policy\s+"([^"]*)"')
# The only font and renderer host this project ever used, by host or by name.
GOOGLE = re.compile(r"google[\s-]*(?:fonts?|operated|hosted)|gstatic|googleapis", re.I)
# A table row (the notice lists recipients as a table) that mentions what the
# web client loads.
CLIENT_LOAD = re.compile(r"font|canvaskit|renderer", re.I)


def csp_sources(nginx):
    """(hosts, wide sources) the CSP lines allow, or None when there is no CSP."""
    lines = CSP.findall(nginx)
    if not lines:
        return None
    hosts, wide = set(), set()
    for line in lines:
        for directive in line.split(";"):
            name, *sources = directive.split() or [""]
            if not (name.endswith("-src") or name == "form-action"):
                continue  # sandbox, report-uri, ... take no host sources
            for src in sources:
                if src.startswith("'"):  # 'self', 'none', 'unsafe-inline', nonces
                    continue
                if src == "*" or re.fullmatch(r"(?:https?|wss?):", src, re.I):
                    wide.add(src)
                elif not (src.endswith(":") and "//" not in src):  # data: blob:
                    host = re.sub(r"^[a-z][a-z0-9+.-]*://", "", src, flags=re.I)
                    host = re.split(r"[/?#]", host)[0]
                    hosts.add(re.sub(r":(?:\d+|\*)$", "", host).lower())
    return hosts, wide


def allows(pattern, host):
    """CSP host-source match: exact, or *.example.com for any subdomain."""
    return host.endswith(pattern[1:]) if pattern.startswith("*.") else host == pattern


def problems(privacy, nginx):
    found = csp_sources(nginx)
    if found is None:
        return ['client/nginx.conf has no `add_header Content-Security-Policy "..."` line']
    hosts, wide = found
    named = {m.group(0).lower() for m in HOST.finditer(privacy)}
    out = [f"the CSP allows every host of {s!r}, which PRIVACY.md cannot name" for s in sorted(wide)]
    out += [
        f"PRIVACY.md names {h}, which the CSP does not allow "
        "(add it to SERVER_SIDE only if the browser never loads from it)"
        for h in sorted(named - SERVER_SIDE)
        if not any(allows(p, h) for p in hosts)
    ]
    out += [
        f"the CSP allows {p}, which PRIVACY.md does not name"
        for p in sorted(hosts)
        if not any(allows(p, h) for h in named)
    ]
    if hosts or wide:
        return out
    # The CSP allows 'self' alone: the notice may not say the client reaches further.
    if GOOGLE.search(privacy):
        out.append("PRIVACY.md describes a Google font or renderer host, but the CSP allows 'self' only")
    out += [
        "PRIVACY.md lists a font or renderer as a recipient, but the CSP allows "
        f"'self' only (reword the row if it is not one): {row.strip()[:70]}"
        for row in privacy.splitlines()
        if row.lstrip().startswith("|") and CLIENT_LOAD.search(row)
    ]
    sentences = re.split(r"(?<=[.!?])\s+", " ".join(privacy.split()))
    if not any(all(k in s.lower() for k in ("canvaskit", "font", "own origin")) for s in sentences):
        out.append(
            "PRIVACY.md must say the fonts and the CanvasKit renderer come from "
            "the deployment's own origin (the CSP allows 'self' only)"
        )
    return out


GSTATIC = "fonts.gstatic.com"
GSTATIC_CSP = (
    'add_header Content-Security-Policy "default-src \'self\'; '
    f"font-src 'self' https://{GSTATIC}\" always;"
)


def regressions(privacy, nginx):
    """name -> (PRIVACY.md, nginx.conf) texts the guard must reject."""
    return {
        "a Google font host named in the notice": (privacy + "\nFonts come from fonts.googleapis.com.\n", nginx),
        "Google Fonts named without a host": (privacy + "\nThe web client loads Google Fonts.\n", nginx),
        "a font CDN listed as a recipient": (privacy + "\n| A font CDN | Your IP address. | Page load. |\n", nginx),
        "the own-origin sentence dropped": (re.sub("own origin", "origin", privacy, flags=re.I), nginx),
        "a CSP host the notice never names": (privacy, GSTATIC_CSP),
        "no CSP line to compare with": (privacy, "server {}"),
    }


def selfproof(privacy, nginx):
    """Each regression the guard exists for must still turn it red."""
    cases = regressions(privacy, nginx)
    out = [f"selfproof: the guard stayed green for {why}" for why, (p, n) in cases.items() if not problems(p, n)]
    if problems(privacy + f"\nFonts load from {GSTATIC}.\n", GSTATIC_CSP):
        out.append("selfproof: the guard rejected a notice that names the host the CSP allows")
    return out


def main(argv):
    privacy_path = pathlib.Path(argv[0]) if argv else ROOT / "PRIVACY.md"
    nginx_path = pathlib.Path(argv[1]) if len(argv) > 1 else ROOT / "client" / "nginx.conf"
    privacy = privacy_path.read_text(encoding="utf-8")
    nginx = nginx_path.read_text(encoding="utf-8")
    failures = problems(privacy, nginx) or selfproof(privacy, nginx)
    for failure in failures:
        print(f"privacy vs CSP: {failure}", file=sys.stderr)
    if failures:
        return 1
    hosts, wide = csp_sources(nginx)
    allowed = ", ".join(sorted(hosts | wide)) or "'self' only"
    cases = len(regressions(privacy, nginx))
    print(f"privacy vs CSP (rv4) ok: the CSP allows {allowed}; the notice agrees; {cases} regressions still go red")
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
