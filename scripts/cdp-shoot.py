#!/usr/bin/env python3
"""cdp-shoot.py — full-page and dark-mode website captures via CDP.

Viewport screenshots (shoot-sites.sh) miss long pages; this tool drives
chromium over the DevTools protocol and captures the whole document
(Page.captureScreenshot with captureBeyondViewport) plus, optionally, a
dark-mode pass (Emulation.setEmulatedMedia prefers-color-scheme=dark).

Uses a minimal stdlib websocket client (no third-party deps; nix python
package envs proved unreliable for `websockets`).

Output naming (feeds the visionreviewd glob):
  <project>/Home--<mode>--<viewport>.png   mode: light|dark

Usage:
  scripts/cdp-shoot.py                       # all sites, home, desktop+mobile, light+dark
  scripts/cdp-shoot.py --viewport desktop    # desktop only
  scripts/cdp-shoot.py --mode dark           # dark pass only
  scripts/cdp-shoot.py --subpages            # also capture the getting-started pages
  scripts/cdp-shoot.py --subpages --skip-home gogenfilter  # subpages only, subset
  scripts/cdp-shoot.py gogenfilter learnings # subset
"""

import base64
import json
import os
import secrets
import shutil
import signal
import socket
import struct
import subprocess
import sys
import tempfile
import time
import urllib.request

CHROME = os.environ.get(
    "VISION_CHROME",
    "/nix/store/lgg6q8vg34lpnm69xxfn8nsj9pvv6jss-chromium-153.0.8010.47/bin/chromium",
)
PORT = 9333
OUT = os.environ.get(
    "VISION_SHOOT_OUT",
    os.path.expanduser("~/.local/share/vision-review-agent/screenshots"),
)
SETTLE_SECONDS = float(os.environ.get("VISION_SHOOT_SETTLE", "2.5"))

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

VIEWPORTS = {
    "desktop": {"width": 1440, "height": 900, "mobile": False},
    "mobile": {"width": 412, "height": 915, "mobile": True},
}

# key docs subpages (getting-started pages where they exist; cmdguard and
# typespec-asyncapi are single-page sites — their sitemaps point at the
# KNOWN-broken custom domains, and src/pages holds only index.astro)
SUBPAGES = {
    "art-dupl": ["getting-started/installation/", "getting-started/quick-start/"],
    "cleanwizard": ["getting-started/installation/", "getting-started/quick-start/"],
    "dynamicmarkdown": [
        "getting-started/installation/",
        "getting-started/quick-start/",
    ],
    "emeet-pixyd": ["getting-started/installation/", "getting-started/quick-start/"],
    "atomicwrite": ["getting-started/installation/", "getting-started/quick-start/"],
    "branded-id": ["getting-started/installation/", "getting-started/quick-start/"],
    "errorfamily": ["getting-started/installation/", "getting-started/quick-start/"],
    "filewatcher": ["getting-started/installation/", "getting-started/quick-start/"],
    "go-output": ["getting-started/installation/", "getting-started/quick-start/"],
    "go-workflow-auditlog": [
        "getting-started/installation/",
        "getting-started/quick-start/",
    ],
    "gogenfilter": ["getting-started/installation/", "getting-started/quick-start/"],
    "md-go-validator": [
        "getting-started/installation/",
        "getting-started/quick-start/",
    ],
    "do-auditlog": ["getting-started/installation/", "getting-started/quick-start/"],
    "templcomponents": ["getting-started/installation", "getting-started/quick-start"],
    "learnings": [
        "docs/getting-started/installation",
        "docs/getting-started/configuration",
    ],
}


class WS:
    """Minimal RFC6455 client: one text message in, one text message out."""

    def __init__(self, url):
        assert url.startswith("ws://"), url
        hostport, path = url[5:].split("/", 1)
        host, port = hostport.split(":")
        self.sock = socket.create_connection((host, int(port)), timeout=60)
        key = base64.b64encode(secrets.token_bytes(16)).decode()
        req = (
            f"GET /{path} HTTP/1.1\r\n"
            f"Host: {hostport}\r\n"
            "Upgrade: websocket\r\n"
            "Connection: Upgrade\r\n"
            f"Sec-WebSocket-Key: {key}\r\n"
            "Sec-WebSocket-Version: 13\r\n\r\n"
        )
        self.sock.sendall(req.encode())
        buf = b""
        while b"\r\n\r\n" not in buf:
            chunk = self.sock.recv(4096)
            if not chunk:
                raise RuntimeError("ws handshake EOF")
            buf += chunk
        head, self.rxbuf = buf.split(b"\r\n\r\n", 1)
        status = head.split(b"\r\n")[0]
        if b"101" not in status:
            raise RuntimeError(f"ws upgrade failed: {status!r}")

    def _read(self, n):
        while len(self.rxbuf) < n:
            chunk = self.sock.recv(1 << 16)
            if not chunk:
                raise EOFError("ws closed")
            self.rxbuf += chunk
        out, self.rxbuf = self.rxbuf[:n], self.rxbuf[n:]
        return out

    def send(self, obj):
        payload = json.dumps(obj).encode()
        mask = secrets.token_bytes(4)
        head = bytearray([0x81])
        n = len(payload)
        if n < 126:
            head.append(0x80 | n)
        elif n < 1 << 16:
            head.append(0x80 | 126)
            head += struct.pack(">H", n)
        else:
            head.append(0x80 | 127)
            head += struct.pack(">Q", n)
        head += mask
        self.sock.sendall(
            bytes(head) + bytes(b ^ mask[i % 4] for i, b in enumerate(payload))
        )

    def recv(self):
        while True:
            msg = self._frame()
            if msg is None:
                continue  # ping/pong/close frames ignored
            return json.loads(msg)

    def _frame(self):
        b1, b2 = self._read(2)
        opcode = b1 & 0x0F
        n = b2 & 0x7F
        if n == 126:
            n = struct.unpack(">H", self._read(2))[0]
        elif n == 127:
            n = struct.unpack(">Q", self._read(8))[0]
        mask = self._read(4) if b2 & 0x80 else b""
        data = self._read(n)
        if mask:
            data = bytes(b ^ mask[i % 4] for i, b in enumerate(data))
        if opcode == 8:
            raise EOFError("ws close frame")
        if opcode not in (1, 2):  # ping (9) / pong (10) etc.
            return None
        return data


class CDP:
    def __init__(self, ws_url):
        self.ws = WS(ws_url)
        self.next_id = 1

    def cmd(self, method, params=None, timeout=60.0):
        mid = self.next_id
        self.next_id += 1
        self.ws.send({"id": mid, "method": method, "params": params or {}})
        deadline = time.time() + timeout
        while time.time() < deadline:
            msg = self.ws.recv()
            if msg.get("id") == mid:
                if "error" in msg:
                    raise RuntimeError(f"{method}: {msg['error']}")
                return msg.get("result", {})
        raise TimeoutError(method)


def http_json(path, method="GET"):
    req = urllib.request.Request(f"http://127.0.0.1:{PORT}{path}", method=method)
    with urllib.request.urlopen(req, timeout=10) as r:
        return json.loads(r.read())


def wait_devtools(deadline=30):
    end = time.time() + deadline
    while time.time() < end:
        try:
            return http_json("/json/version")
        except Exception:
            time.sleep(0.3)
    raise TimeoutError("devtools endpoint never came up")


def page_label(url_path):
    """docs/getting-started/installation/ -> GettingStartedInstallation"""
    parts = [p for p in url_path.split("/") if p]
    return "".join(p[:1].upper() + p[1:] for p in parts)[:60]


def shoot(
    project, url, modes, viewports, user_data_dir, subpage_paths=(), skip_home=False
):
    results = []
    pages = [] if skip_home else [("Home", url)]
    for sp in subpage_paths:
        full = url.rstrip("/") + "/" + sp.lstrip("/")
        pages.append((page_label(sp), full))
    if not pages:
        print(f"  skip {project}: no pages selected (single-page site?)")
        return results
    for vp_name in viewports:
        vp = VIEWPORTS[vp_name]
        target = http_json("/json/new?about:blank", method="PUT")
        ws_url = target["webSocketDebuggerUrl"]
        cdp = CDP(ws_url)
        try:
            cdp.cmd("Page.enable")
            cdp.cmd("Runtime.enable")
            cdp.cmd(
                "Emulation.setDeviceMetricsOverride",
                {
                    "width": vp["width"],
                    "height": vp["height"],
                    "deviceScaleFactor": 1,
                    "mobile": vp["mobile"],
                },
            )
            for mode in modes:
                if mode == "dark":
                    cdp.cmd(
                        "Emulation.setEmulatedMedia",
                        {
                            "features": [
                                {"name": "prefers-color-scheme", "value": "dark"}
                            ]
                        },
                    )
                else:
                    cdp.cmd(
                        "Emulation.setEmulatedMedia",
                        {
                            "features": [
                                {"name": "prefers-color-scheme", "value": "light"}
                            ]
                        },
                    )
                for label, page_url in pages:
                    for attempt in (1, 2):
                        try:
                            cdp.cmd("Page.navigate", {"url": page_url})
                            loaded = False
                            deadline = time.time() + 45
                            while time.time() < deadline and not loaded:
                                msg = cdp.ws.recv()
                                if msg.get("method") == "Page.loadEventFired":
                                    loaded = True
                            time.sleep(SETTLE_SECONDS)
                            ready = cdp.cmd(
                                "Runtime.evaluate",
                                {
                                    "expression": "document.readyState",
                                    "returnByValue": True,
                                },
                            )
                            if ready.get("result", {}).get("value") != "complete":
                                time.sleep(3)
                            height = cdp.cmd(
                                "Runtime.evaluate",
                                {
                                    "expression": "Math.max(document.documentElement.scrollHeight,"
                                    " document.body ? document.body.scrollHeight : 0)",
                                    "returnByValue": True,
                                },
                            )["result"]["value"]
                            height = max(int(height or 0), vp["height"])
                            shot = cdp.cmd(
                                "Page.captureScreenshot",
                                {
                                    "format": "png",
                                    "captureBeyondViewport": True,
                                    "clip": {
                                        "x": 0,
                                        "y": 0,
                                        "width": vp["width"],
                                        "height": height,
                                        "scale": 1,
                                    },
                                },
                                timeout=120,
                            )
                            break
                        except TimeoutError as e:
                            # one retry: a hung capture must not fail the
                            # whole site; a second timeout bubbles up
                            if attempt == 2:
                                raise
                            print(
                                f"  retry {project}/{label}--{mode}--{vp_name}"
                                f" (attempt {attempt} timed out: {e})"
                            )
                    data = base64.b64decode(shot["data"])
                    outdir = os.path.join(OUT, project)
                    os.makedirs(outdir, exist_ok=True)
                    name = f"{label}--{mode}--{vp_name}.png"
                    path = os.path.join(outdir, name)
                    with open(path, "wb") as f:
                        f.write(data)
                    results.append((name, len(data), height))
                    print(f"  shot {project}/{name} ({len(data)}B, docH={height})")
        finally:
            try:
                cdp.ws.sock.close()
            except Exception:
                pass
            try:
                http_json(f"/json/close/{target['id']}")
            except Exception:
                pass
    return results


def main():
    args = sys.argv[1:]
    modes, viewports, filt = ["light", "dark"], ["desktop", "mobile"], []
    subpages = skip_home_flag = False
    while args:
        a = args.pop(0)
        if a == "--mode":
            modes = args.pop(0).split(",")
        elif a == "--viewport":
            viewports = args.pop(0).split(",")
        elif a == "--subpages":
            subpages = True
        elif a == "--skip-home":
            skip_home_flag = True
        elif a.startswith("--"):
            print(f"unknown flag {a}", file=sys.stderr)
            return 2
        else:
            filt.append(a)

    if shutil.which(CHROME) is None and not os.path.exists(CHROME):
        print(f"chromium not found: {CHROME}", file=sys.stderr)
        return 1

    user_data_dir = tempfile.mkdtemp(prefix="cdp-shoot-")
    proc = subprocess.Popen(
        [
            CHROME,
            "--headless",
            "--no-sandbox",
            "--disable-gpu",
            "--hide-scrollbars",
            "--blink-settings=lazyImageLoadingEnabled=false",
            f"--remote-debugging-port={PORT}",
            f"--user-data-dir={user_data_dir}",
            "about:blank",
        ],
        stdout=subprocess.DEVNULL,
        stderr=subprocess.DEVNULL,
    )
    fail = 0
    total_shots = 0
    try:
        wait_devtools()
        for proj, url in SITES:
            if filt and proj not in filt:
                continue
            print(f"=== {proj} ({url})")
            try:
                sp = tuple(SUBPAGES.get(proj, ())) if subpages else ()
                expected = (
                    ((0 if skip_home_flag else 1) + len(sp))
                    * len(modes)
                    * len(viewports)
                )
                results = shoot(
                    proj, url, modes, viewports, user_data_dir, sp, skip_home_flag
                )
                total_shots += len(results)
                if len(results) != expected:
                    print(
                        f"  INVENTORY-FAIL {proj}: captured {len(results)} shots,"
                        f" expected {expected} (pages x modes x viewports)"
                    )
                    fail += 1
                for name, size, doc_h in results:
                    if size < 8000:
                        print(f"  VERIFY-FAIL {proj}/{name}: only {size}B")
                        fail += 1
            except Exception as e:
                print(f"  FAIL {proj}: {e}")
                fail += 1
    finally:
        proc.terminate()
        try:
            proc.wait(timeout=5)
        except subprocess.TimeoutExpired:
            proc.kill()
        signal.signal(signal.SIGALRM, signal.SIG_DFL)
        subprocess.run(["rm", "-rf", user_data_dir], check=False)
    print(f"DONE: {total_shots} shots, {fail} failures")
    return 1 if fail else 0


if __name__ == "__main__":
    sys.exit(main())
