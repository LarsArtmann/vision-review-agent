#!/usr/bin/env python3
"""fleet_audit2.py — light header/link audits across the 17-site fleet.

Covers four NEXT-list items in one pass:
  #12 canonical URL presence + origin match (go-workflow-auditlog parity)
  #11 CSP header presence/type after the og/meta additions
  #39 hreflang alternates (single-locale sites: expect none -> close)
  #40 favicon: declared icon links + /favicon.ico fallback status
"""

import json
import re
import ssl
import sys
import urllib.request

SITES = [
    ("art-dupl", "https://art-dupl.lars.software"),
    ("cleanwizard", "https://cleanwizard.lars.software"),
    ("cmdguard", "https://cmdguard.web.app"),
    ("dynamicmarkdown", "https://dynamicmarkdown.lars.software"),
    ("emeet-pixyd", "https://emeet-pixyd.lars.software"),
    ("atomicwrite", "https://atomicwrite.lars.software"),
    ("branded-id", "https://branded-id.lars.software"),
    ("errorfamily", "https://errorfamily.lars.software"),
    ("filewatcher", "https://filewatcher.lars.software"),
    ("go-output", "https://go-output.lars.software"),
    ("go-workflow-auditlog", "https://go-workflow-auditlog.lars.software"),
    ("gogenfilter", "https://gogenfilter.lars.software"),
    ("md-go-validator", "https://md-go-validator.lars.software"),
    ("do-auditlog", "https://do-auditlog.lars.software"),
    ("typespec-asyncapi", "https://typespec-asyncapi.web.app"),
    ("templcomponents", "https://templcomponents.lars.software"),
    ("learnings", "https://lars-learnings.web.app"),
]

CTX = ssl.create_default_context()
UA = {"User-Agent": "Mozilla/5.0 (X11; Linux x86_64) fleet-audit/2"}


def fetch(url, method="GET"):
    req = urllib.request.Request(url, headers=UA, method=method)
    try:
        with urllib.request.urlopen(req, timeout=15, context=CTX) as r:
            return r.status, dict(r.headers), r.read(400_000).decode("utf-8", "replace")
    except urllib.error.HTTPError as e:
        return e.code, dict(e.headers), ""
    except Exception as e:
        return 0, {}, str(e)


def meta_links(html):
    return re.findall(
        r'<link\b[^>]*rel="([^"]*)"[^>]*href="([^"]*)"[^>]*>', html, re.IGNORECASE
    ) + re.findall(
        r'<link\b[^>]*href="([^"]*)"[^>]*rel="([^"]*)"[^>]*>', html, re.IGNORECASE
    )


def audit(name, base):
    row = {"site": name}
    status, headers, body = fetch(base + "/")
    row["home_status"] = status
    if status != 200:
        row["error"] = "home fetch failed"
        return row

    csp = headers.get("Content-Security-Policy")
    csp_ro = headers.get("Content-Security-Policy-Report-Only")
    row["csp"] = "enforced" if csp else ("report-only" if csp_ro else "none")
    if csp_ro:
        row["csp_ro_manifest_ok"] = (
            "manifest-src" in csp_ro or "'self'" in csp_ro or "default-src" in csp_ro
        )

    pairs = meta_links(body)
    rels = {}
    for a, b in pairs:
        rel, href = (
            (a.lower(), b)
            if a.lower()
            in ("canonical", "alternate", "icon", "manifest", "apple-touch-icon")
            else (b.lower(), a)
        )
        if rel in rels and rel != "alternate":
            rels[rel] += "|" + href
        elif rel in ("canonical", "alternate", "icon", "manifest", "apple-touch-icon"):
            rels[rel] = href

    can = rels.get("canonical", "")
    row["canonical"] = (
        "missing" if not can else ("ok" if can.startswith(base) else f"MISMATCH:{can}")
    )
    row["og_url"] = "ok" if 'property="og:url"' in body or "og:url" in body else "none"

    hl = [h for h in re.findall(r'hreflang="([^"]*)"', body)]
    row["hreflang"] = ",".join(hl) if hl else "none"

    icon = rels.get("icon", "")
    row["icon_link"] = icon if icon else "missing"
    ico_status, _, _ = fetch(base + "/favicon.ico", method="HEAD")
    row["favicon_ico"] = ico_status
    man = rels.get("manifest", "")
    row["manifest"] = "ok" if man else "missing"
    return row


def main():
    out = []
    for name, base in SITES:
        row = audit(name, base)
        out.append(row)
        print(json.dumps(row), flush=True)
    with open("/tmp/vra/fleet_audit2.json", "w") as f:
        json.dump(out, f, indent=1)

    print("\n=== summary ===", file=sys.stderr)
    for r in out:
        if "error" in r:
            print(f"{r['site']}: ERROR {r['error']}")
            continue
        print(
            f"{r['site']}: canonical={r['canonical']} csp={r['csp']}"
            f" hreflang={r['hreflang']} ico={r['favicon_ico']}"
            f" manifest={r['manifest']} icon={r['icon_link']}"
        )


if __name__ == "__main__":
    main()
