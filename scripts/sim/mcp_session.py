#!/usr/bin/env python3
"""Drive the insideout MCP server over stdio for user simulations.

Usage:
  mcp_session.py '[{"tool":"roadmap_list","args":{"project":"<id>"}}, ...]'

Reads a JSON array of {tool, args} calls, spawns one insideout-mcp
process (env: INSIDEOUT_API, optional INSIDEOUT_TOKEN), performs the
initialize handshake, lists tools when asked (tool: "__list__"), and
prints one labeled JSON result per call. Process-login auth (auth_*
tools) works because every call rides the same MCP process.

Simulation data only — this driver prints whatever the tools return.
"""

import json
import os
import subprocess
import sys

MCP_BIN = os.environ.get(
    "INSIDEOUT_MCP_BIN",
    os.path.join(os.path.dirname(__file__), "..", "..", "server",
                 "cmd", "insideout-mcp", "insideout-mcp"),
)


class McpSession:
    def __init__(self):
        self.proc = subprocess.Popen(
            [MCP_BIN], stdin=subprocess.PIPE, stdout=subprocess.PIPE,
            stderr=subprocess.DEVNULL, text=True, bufsize=1,
        )
        self._id = 0

    def _send(self, method, params):
        self._id += 1
        req = {"jsonrpc": "2.0", "id": self._id, "method": method,
               "params": params}
        self.proc.stdin.write(json.dumps(req) + "\n")
        self.proc.stdin.flush()
        while True:
            line = self.proc.stdout.readline()
            if not line:
                raise RuntimeError("MCP process closed stdio")
            msg = json.loads(line)
            if msg.get("id") == self._id:
                return msg

    def initialize(self):
        r = self._send("initialize", {
            "protocolVersion": "2024-11-05",
            "capabilities": {},
            "clientInfo": {"name": "insideout-sim", "version": "1.0"},
        })
        self._send("notifications/initialized", {})
        return r.get("result", {}).get("serverInfo", {})

    def call(self, name, args):
        r = self._send("tools/call", {"name": name, "arguments": args or {}})
        result = r.get("result", {})
        if result.get("isError"):
            return {"error": result.get("content")}
        content = result.get("content", [])
        texts = [c.get("text", "") for c in content if c.get("type") == "text"]
        joined = "\n".join(texts)
        try:
            return json.loads(joined)
        except (ValueError, TypeError):
            return {"raw": joined}

    def close(self):
        try:
            self.proc.stdin.close()
            self.proc.wait(timeout=3)
        except Exception:
            self.proc.kill()


def main() -> int:
    calls = json.loads(sys.argv[1] if len(sys.argv) > 1 else "[]")
    s = McpSession()
    info = s.initialize()
    print(f"# session: {info.get('name')} {info.get('version')}")
    for i, c in enumerate(calls, 1):
        name = c.get("tool")
        if name == "__list__":
            tools = s._send("tools/list", {}).get("result", {}).get("tools", [])
            out = [t["name"] for t in tools]
        else:
            out = s.call(name, c.get("args"))
        print(f"== [{i}] {name}")
        print(json.dumps(out, indent=2, ensure_ascii=False, default=str)[:4000])
    s.close()
    return 0


if __name__ == "__main__":
    sys.exit(main())
