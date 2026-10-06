#!/usr/bin/env python3
"""Minimal CDP driver for user simulations against the Flutter web
client (CanvasKit renders one canvas — automation is vision-driven:
screenshot, read coordinates, click/type).

One-line verbs (Chrome stays up between calls):
  web_driver.py launch              # start headless Chrome on :9222
  web_driver.py launch-ui           # visible Chrome with DevTools open
  web_driver.py nav <url>
  web_driver.py shot <png-path>
  web_driver.py click <x> <y>
  web_driver.py type <text>         # types into the focused field
  web_driver.py press <key>         # Enter, Tab, Backspace
  web_driver.py url                 # current page URL
  web_driver.py eval '<js>'         # Runtime.evaluate, prints the value
  web_driver.py console             # buffered console messages (needs logs on)
  web_driver.py quit

Simulation only; nothing here touches real accounts.
"""

import base64
import json
import subprocess
import sys
import time
import urllib.request

import websocket

CDP_PORT = 9222
CHROME = "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"
PROFILE = "/tmp/insideout-sim/chrome-profile"
KEYS = {"Enter": 13, "Tab": 9, "Backspace": 8, "Escape": 27}


def _targets():
    with urllib.request.urlopen(f"http://127.0.0.1:{CDP_PORT}/json") as r:
        return json.load(r)


def _page_ws():
    for t in _targets():
        if t.get("type") == "page":
            return t["webSocketDebuggerUrl"]
    raise RuntimeError("no page target — run: web_driver.py launch")


class Cdp:
    def __init__(self):
        self.ws = websocket.create_connection(_page_ws(), timeout=60)
        self._id = 0

    def send(self, method, params=None):
        self._id += 1
        self.ws.send(json.dumps({"id": self._id, "method": method,
                                 "params": params or {}}))
        while True:
            msg = json.loads(self.ws.recv())
            if msg.get("id") == self._id:
                if "error" in msg:
                    raise RuntimeError(f"{method}: {msg['error']}")
                return msg.get("result", {})

    def evaluate(self, expr):
        r = self.send("Runtime.evaluate", {"expression": expr,
                                           "returnByValue": True})
        return r.get("result", {}).get("value")

    def navigate(self, url):
        self.send("Page.navigate", {"url": url})
        time.sleep(4)  # Flutter debug build boot

    def logs_on(self):
        self.send("Runtime.enable")
        self.send("Log.enable")

    def reload_and_collect(self, seconds=20):
        """Reload the page and drain console/log events for a while."""
        buf = []
        self.logs_on()
        self._id += 1
        self.ws.send(json.dumps({"id": self._id, "method": "Page.reload"}))
        deadline = time.time() + seconds
        self.ws.settimeout(2)
        while time.time() < deadline:
            try:
                msg = json.loads(self.ws.recv())
            except websocket.WebSocketTimeoutException:
                continue
            entry = None
            if msg.get("method") == "Runtime.consoleAPICalled":
                args = " ".join(str(a.get("value", a.get("description", "")))
                                for a in msg["params"].get("args", []))
                entry = f"[{msg['params'].get('type')}] {args}"
            elif msg.get("method") == "Log.entryAdded":
                p = msg["params"]["entry"]
                entry = f"[{p.get('level')}] {p.get('text')} {p.get('url', '')}"
            elif msg.get("method") == "Runtime.exceptionThrown":
                d = msg["params"]["exceptionDetails"]
                entry = f"[exception] {d.get('text')} {d.get('exception', {}).get('description', '')}"
            if entry:
                buf.append(entry[:500])
        return buf

    def screenshot(self, path):
        data = self.send("Page.captureScreenshot", {"format": "png"})["data"]
        with open(path, "wb") as fh:
            fh.write(base64.b64decode(data))

    def click(self, x, y):
        for etype in ("mousePressed", "mouseReleased"):
            self.send("Input.dispatchMouseEvent", {
                "type": etype, "x": x, "y": y, "button": "left",
                "clickCount": 1,
            })
        time.sleep(0.8)

    def type(self, text):
        self.send("Input.insertText", {"text": text})
        time.sleep(0.3)

    def press(self, key):
        code = KEYS.get(key)
        if not code:
            raise RuntimeError(f"unsupported key {key}")
        args = {"type": "keyDown", "key": key, "windowsVirtualKeyCode": code,
                "nativeVirtualKeyCode": code}
        self.send("Input.dispatchKeyEvent", args)
        args["type"] = "keyUp"
        self.send("Input.dispatchKeyEvent", args)
        time.sleep(0.5)


def launch(headed=False):
    args = [
        CHROME, f"--remote-debugging-port={CDP_PORT}",
        f"--user-data-dir={PROFILE}",
        "--window-size=1440,900", "--no-first-run", "--no-default-browser-check",
        "--remote-allow-origins=*",
        # The app's boot gate awaits document.fonts.load(), which pends
        # forever when fonts.gstatic.com is unreachable (see docs/issues).
        "--disable-remote-fonts",
    ]
    if headed:
        args.append("--auto-open-devtools-for-tabs")
    else:
        args.append("--headless=new")
    args.append("about:blank")
    subprocess.Popen(args, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
    for _ in range(50):
        try:
            _targets()
            print("chrome up on :9222")
            return
        except Exception:
            time.sleep(0.2)
    raise RuntimeError("chrome did not start")


def main() -> int:
    if len(sys.argv) < 2:
        print(__doc__)
        return 2
    verb, args = sys.argv[1], sys.argv[2:]
    if verb == "launch":
        launch()
        return 0
    if verb == "launch-ui":
        launch(headed=True)
        return 0
    if verb == "quit":
        subprocess.run(["pkill", "-f", f"user-data-dir={PROFILE}"])
        print("chrome down")
        return 0
    c = Cdp()
    if verb == "nav":
        c.navigate(args[0])
        print("navigated")
    elif verb == "shot":
        c.screenshot(args[0])
        print(f"shot {args[0]}")
    elif verb == "click":
        c.click(float(args[0]), float(args[1]))
        print(f"clicked {args[0]},{args[1]}")
    elif verb == "type":
        c.type(args[0])
        print(f"typed {len(args[0])} chars")
    elif verb == "press":
        c.press(args[0])
        print(f"pressed {args[0]}")
    elif verb == "url":
        print(c.evaluate("location.href"))
    elif verb == "eval":
        print(c.evaluate(args[0]))
    elif verb == "console":
        for line in c.reload_and_collect():
            print(line)
    else:
        print(f"unknown verb {verb}")
        return 2
    return 0


if __name__ == "__main__":
    sys.exit(main())
