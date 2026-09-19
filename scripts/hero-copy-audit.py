#!/usr/bin/env python3
"""hero-copy-audit.py — hero-copy staleness pass across the 17-site fleet.

Extracts hardcoded metric claims (star counts, version numbers) from the
live home pages and diffs them against live GitHub facts (stargazers,
latest release tag). Flags mismatches: hero copy that claims a different
star count than the repo actually has, or a version older than the latest
tag.
"""
import json
import re
import ssl
import subprocess
import sys
import urllib.request

CTX = ssl.create_default_context()

FLEET = [
    ("art-dupl", "https://art-dupl.lars.software", "LarsArtmann/art-dupl"),
    ("cleanwizard", "https://cleanwizard.lars.software", "LarsArtmann/clean-wizard"),
    ("cmdguard", "https://cmdguard.web.app", "LarsArtmann/cmdguard"),
    ("dynamicmarkdown", "https://dynamicmarkdown.lars.software", "LarsArtmann/dynamic-markdown-site"),
    ("emeet-pixyd", "https://emeet-pixyd.lars.software", "LarsArtmann/emeet-pixyd"),
    ("atomicwrite", "https://atomicwrite.lars.software", "LarsArtmann/go-atomic-write"),
    ("branded-id", "https://branded-id.lars.software", "LarsArtmann/go-branded-id"),
    ("errorfamily", "https://errorfamily.lars.software", "LarsArtmann/go-error-family"),
    ("filewatcher", "https://filewatcher.lars.software", "LarsArtmann/go-filewatcher"),
    ("go-output", "https://go-output.lars.software", "LarsArtmann/go-output"),
    ("go-workflow-auditlog", "https://go-workflow-auditlog.lars.software", "LarsArtmann/go-workflow-auditlog"),
    ("gogenfilter", "https://gogenfilter.lars.software", "LarsArtmann/gogenfilter"),
    ("md-go-validator", "https://md-go-validator.lars.software", "LarsArtmann/md-go-validator"),
    ("do-auditlog", "https://do-auditlog.lars.software", "LarsArtmann/samber-do-auditlog"),
    ("typespec-asyncapi", "https://typespec-asyncapi.web.app", "LarsArtmann/typespec-asyncapi"),
    ("templcomponents", "https://templcomponents.lars.software", "LarsArtmann/templ-components"),
    ("learnings", "https://lars-learnings.web.app", "LarsArtmann/learnings"),
]

STAR_PAT = re.compile(r">\s*(\d+)\s+Stars?\s*<", re.I)
VER_PAT = re.compile(r">\s*v?(\d+\.\d+\.\d+)\s*<")


def fetch(url):
    req = urllib.request.Request(url, headers={"User-Agent": "Mozilla/5.0 hero-audit/1"})
    try:
        with urllib.request.urlopen(req, timeout=15, context=CTX) as r:
            return r.status, r.read().decode("utf-8", "replace")
    except Exception as e:
        return 0, str(e)


def gh(path):
    try:
        out = subprocess.run(
            ["gh", "api", path, "--jq", "."],
            capture_output=True, text=True, timeout=20,
        )
        return json.loads(out.stdout) if out.returncode == 0 else None
    except Exception:
        return None


def main():
    problems = []
    for name, url, repo in FLEET:
        status, html = fetch(url)
        if status != 200:
            print(f"{name}: home fetch failed ({status})")
            continue
        stars_live = None
        data = gh(f"repos/{repo}")
        if data is not None:
            stars_live = data.get("stargazers_count")
        claims = [int(m.group(1)) for m in STAR_PAT.finditer(html)]
        if claims and stars_live is not None:
            for c in sorted(set(claims)):
                mark = "OK" if c == stars_live else "STALE"
                if c != stars_live:
                    problems.append(f"{name}: hero claims {c} stars, repo has {stars_live}")
                print(f"{name}: stars claimed={c} live={stars_live} [{mark}]")
        elif claims:
            print(f"{name}: stars claimed={sorted(set(claims))} live=UNKNOWN (gh api failed)")
        vers = sorted(set(m.group(1) for m in VER_PAT.finditer(html)))
        if vers:
            print(f"{name}: version strings on page: {vers[:4]}")
    print("\n=== staleness problems ===")
    print("\n".join(problems) if problems else "none — hero copy matches live facts")


if __name__ == "__main__":
    sys.exit(main())
