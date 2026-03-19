import { useState, useEffect, useCallback, useRef } from "react";

const INITIAL_CONTAINERS = [
  { id: "web-1", name: "web-1", image: "nginx:latest", status: "running", color: "#22c55e" },
  { id: "api-1", name: "api-1", image: "node:18-alpine", status: "running", color: "#3b82f6" },
  { id: "db-1", name: "db-1", image: "postgres:15", status: "running", color: "#f59e0b" },
  { id: "redis-1", name: "redis-1", image: "redis:7-alpine", status: "running", color: "#ef4444" },
  { id: "worker-1", name: "worker-1", image: "python:3.11", status: "running", color: "#a855f7" },
];

const LOG_TEMPLATES = {
  "web-1": [
    { level: "INFO", msg: '172.18.0.1 - - "GET /api/health HTTP/1.1" 200 15' },
    { level: "INFO", msg: '172.18.0.1 - - "GET /static/app.js HTTP/1.1" 304 0' },
    { level: "WARN", msg: "upstream timed out (110: Connection timed out) while reading response header" },
    { level: "INFO", msg: '172.18.0.1 - - "POST /api/users HTTP/1.1" 201 89' },
    { level: "ERROR", msg: "connect() failed (111: Connection refused) while connecting to upstream" },
    { level: "INFO", msg: '172.18.0.1 - - "GET /dashboard HTTP/1.1" 200 4521' },
  ],
  "api-1": [
    { level: "INFO", msg: "[express] Request handled: GET /api/users (23ms)" },
    { level: "DEBUG", msg: "[pg] Query executed: SELECT * FROM users WHERE active = true (12ms)" },
    { level: "INFO", msg: "[express] Request handled: POST /api/orders (145ms)" },
    { level: "WARN", msg: "[redis] Connection pool running low: 2/10 available" },
    { level: "ERROR", msg: "[express] Unhandled error in /api/payments: ECONNREFUSED" },
    { level: "INFO", msg: "[ws] Client connected: session_abc123" },
    { level: "DEBUG", msg: "[cache] Cache miss for key: user:42:profile" },
  ],
  "db-1": [
    { level: "INFO", msg: "checkpoint starting: time" },
    { level: "INFO", msg: "checkpoint complete: wrote 847 buffers (5.2%); 0 WAL file(s)" },
    { level: "WARN", msg: 'could not open statistics file "pg_stat_tmp/global.stat": Operation not permitted' },
    { level: "INFO", msg: 'automatic vacuum of table "app.public.sessions": removed 1204 dead tuples' },
    { level: "DEBUG", msg: "duration: 234.12 ms  statement: SELECT * FROM orders WHERE created_at > now() - interval '1 hour'" },
    { level: "ERROR", msg: "deadlock detected: Process 142 waits for ShareLock on transaction 9847" },
  ],
  "redis-1": [
    { level: "INFO", msg: "DB 0: 15234 keys (0 volatile) in 16384 slots HT." },
    { level: "WARN", msg: "Memory usage exceeds 75%: used 384MB / max 512MB" },
    { level: "INFO", msg: "RDB: 0 MB of memory used by copy-on-write" },
    { level: "DEBUG", msg: "Client closed connection: id=847 addr=172.18.0.3:44210" },
    { level: "INFO", msg: "Background saving started by pid 42" },
    { level: "INFO", msg: "Background saving terminated with success" },
  ],
  "worker-1": [
    { level: "INFO", msg: "[celery] Task email.send_welcome[a1b2c3] succeeded in 0.342s" },
    { level: "INFO", msg: "[celery] Received task: report.generate[d4e5f6]" },
    { level: "WARN", msg: "[celery] Task report.generate[d4e5f6] retry 2/3: ConnectionError" },
    { level: "ERROR", msg: "[celery] Task report.generate[d4e5f6] failed: MaxRetriesExceeded" },
    { level: "INFO", msg: "[celery] Task order.process[g7h8i9] succeeded in 1.205s" },
    { level: "DEBUG", msg: "[celery] Worker heartbeat: 5 active, 12 reserved, 847 processed" },
  ],
};

const SHELL_LINES = {
  "web-1": [
    "root@web-1:/# ",
    "root@web-1:/# nginx -t",
    "nginx: the configuration file /etc/nginx/nginx.conf syntax is ok",
    "nginx: configuration file /etc/nginx/nginx.conf test is successful",
    "root@web-1:/# cat /etc/nginx/conf.d/default.conf",
    "server {",
    "    listen       80;",
    "    server_name  localhost;",
    "    location / { proxy_pass http://api-1:3000; }",
    "}",
    "root@web-1:/# ",
  ],
  "api-1": [
    "node@api-1:/app$ ",
    "node@api-1:/app$ node --version",
    "v18.19.0",
    "node@api-1:/app$ ls -la",
    "total 248",
    "drwxr-xr-x 1 node node  4096 Mar 19 10:00 .",
    "-rw-r--r-- 1 node node  1842 Mar 19 09:55 package.json",
    "drwxr-xr-x 1 node node  4096 Mar 19 09:58 node_modules",
    "drwxr-xr-x 1 node node  4096 Mar 19 09:55 src",
    "node@api-1:/app$ ",
  ],
  "db-1": [
    "postgres@db-1:/$ ",
    "postgres@db-1:/$ psql -U app -d app_production",
    "psql (15.4)",
    'Type "help" for help.',
    "",
    "app_production=# \\dt",
    "              List of relations",
    " Schema |    Name    | Type  | Owner",
    "--------+------------+-------+-------",
    " public | users      | table | app",
    " public | orders     | table | app",
    " public | sessions   | table | app",
    "(3 rows)",
    "",
    "app_production=# ",
  ],
  "redis-1": [
    "root@redis-1:/data# ",
    "root@redis-1:/data# redis-cli",
    "127.0.0.1:6379> INFO memory",
    "# Memory",
    "used_memory:402653184",
    "used_memory_human:384.00M",
    "maxmemory:536870912",
    "maxmemory_human:512.00M",
    "127.0.0.1:6379> DBSIZE",
    "(integer) 15234",
    "127.0.0.1:6379> ",
  ],
  "worker-1": [
    "celery@worker-1:/app$ ",
    "celery@worker-1:/app$ celery -A app inspect active",
    "-> worker-1@worker-1: OK",
    "    - email.send_welcome[a1b2c3]: args=('user@test.com',)",
    "    - order.process[j0k1l2]: args=(4521,)",
    "",
    "celery@worker-1:/app$ celery -A app inspect stats | head -5",
    "-> worker-1@worker-1: OK",
    '    {"pool": {"max-concurrency": 8, "processes": [42,43,44,45]}}',
    "celery@worker-1:/app$ ",
  ],
};

const LEVEL_COLORS = {
  INFO: "#9ca3af",
  DEBUG: "#6b7280",
  WARN: "#f59e0b",
  ERROR: "#ef4444",
};

const STATUS_ICONS = {
  running: { icon: "▸", color: "#3fb950" },
  paused: { icon: "⏸", color: "#f0883e" },
  stopped: { icon: "■", color: "#f85149" },
};

function generateTimestamp(offset) {
  const d = new Date(Date.now() - offset * 1000);
  const h = String(d.getHours()).padStart(2, "0");
  const m = String(d.getMinutes()).padStart(2, "0");
  const s = String(d.getSeconds()).padStart(2, "0");
  const ms = String(d.getMilliseconds()).padStart(3, "0");
  return `${h}:${m}:${s}.${ms}`;
}

function generateLogs(count = 80) {
  const logs = [];
  for (let i = count; i >= 0; i--) {
    const c = INITIAL_CONTAINERS[Math.floor(Math.random() * INITIAL_CONTAINERS.length)];
    const templates = LOG_TEMPLATES[c.id];
    const t = templates[Math.floor(Math.random() * templates.length)];
    logs.push({ id: i, container: c, timestamp: generateTimestamp(i * 0.8), level: t.level, message: t.msg });
  }
  return logs;
}

function tryRegex(pattern) {
  try {
    return new RegExp(pattern, "i");
  } catch {
    return null;
  }
}

function highlightMatches(text, regex) {
  if (!regex) return text;
  const parts = [];
  let last = 0;
  let match;
  const r = new RegExp(regex.source, "gi");
  while ((match = r.exec(text)) !== null) {
    if (match.index > last) parts.push(text.slice(last, match.index));
    parts.push(
      <span key={match.index} style={{ background: "#f0883e44", color: "#f0883e", borderRadius: 2 }}>
        {match[0]}
      </span>
    );
    last = match.index + match[0].length;
    if (match[0].length === 0) break;
  }
  if (last < text.length) parts.push(text.slice(last));
  return parts.length > 0 ? parts : text;
}

// Shell component
function ShellPanel({ container, onClose }) {
  const [input, setInput] = useState("");
  const [history, setHistory] = useState(() => SHELL_LINES[container.id] || [`root@${container.name}:/# `]);
  const [cmdHistory, setCmdHistory] = useState([]);
  const [cmdIdx, setCmdIdx] = useState(-1);
  const scrollRef = useRef(null);
  const inputRef = useRef(null);

  useEffect(() => {
    if (scrollRef.current) scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
  }, [history]);

  const prompt = history[history.length - 1] || `root@${container.name}:/# `;

  const handleSubmit = (e) => {
    if (e.key === "Enter" && input.trim()) {
      e.preventDefault();
      e.stopPropagation();
      const responses = {
        ls: "bin  boot  dev  etc  home  lib  media  mnt  opt  proc  root  run  sbin  srv  sys  tmp  usr  var",
        pwd: "/",
        whoami: "root",
        hostname: container.name,
        date: new Date().toString(),
        uptime: " 10:23:45 up 3 days, 2:15, 0 users, load average: 0.42, 0.38, 0.35",
        "cat /etc/os-release": 'PRETTY_NAME="Debian GNU/Linux 12 (bookworm)"',
        env: `HOSTNAME=${container.name}\nPATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin\nHOME=/root`,
        ps: "PID TTY      TIME CMD\n  1 ?        0:05 " + container.image.split(":")[0] + "\n 42 pts/0    0:00 bash\n 89 pts/0    0:00 ps",
        "free -h": "              total    used    free   shared  buff/cache  available\nMem:          512Mi   234Mi    78Mi    12Mi      200Mi      245Mi",
      };
      const newLines = [...history];
      newLines[newLines.length - 1] = prompt + input;
      const resp = responses[input.trim()] || responses[input.trim().split(" ")[0]] || `bash: ${input.trim().split(" ")[0]}: command not found`;
      resp.split("\n").forEach((l) => newLines.push(l));
      newLines.push(prompt.replace(/[^ ]*$/, "") || `root@${container.name}:/# `);
      setHistory(newLines);
      setCmdHistory((h) => [input, ...h]);
      setCmdIdx(-1);
      setInput("");
    } else if (e.key === "ArrowUp") {
      e.preventDefault();
      e.stopPropagation();
      if (cmdHistory.length > 0) {
        const newIdx = Math.min(cmdIdx + 1, cmdHistory.length - 1);
        setCmdIdx(newIdx);
        setInput(cmdHistory[newIdx]);
      }
    } else if (e.key === "ArrowDown") {
      e.preventDefault();
      e.stopPropagation();
      if (cmdIdx > 0) {
        setCmdIdx(cmdIdx - 1);
        setInput(cmdHistory[cmdIdx - 1]);
      } else {
        setCmdIdx(-1);
        setInput("");
      }
    }
  };

  return (
    <div
      style={{ display: "flex", flexDirection: "column", height: "100%", background: "#0d1117" }}
      onClick={() => inputRef.current?.focus()}
    >
      <div ref={scrollRef} style={{ flex: 1, overflow: "auto", padding: "6px 12px", fontSize: 12, lineHeight: "18px" }}>
        {history.slice(0, -1).map((line, i) => (
          <div key={i} style={{ color: "#c9d1d9", whiteSpace: "pre-wrap", wordBreak: "break-all" }}>
            {line}
          </div>
        ))}
        <div style={{ display: "flex", color: "#c9d1d9", whiteSpace: "pre" }}>
          <span>{prompt}</span>
          <input
            ref={inputRef}
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={handleSubmit}
            style={{
              background: "transparent",
              border: "none",
              color: "#c9d1d9",
              fontFamily: "inherit",
              fontSize: "inherit",
              outline: "none",
              flex: 1,
              padding: 0,
              caretColor: "#58a6ff",
            }}
            autoFocus
          />
        </div>
      </div>
    </div>
  );
}

// Action menu component
function ActionMenu({ container, position, onAction, onClose }) {
  const actions = container.status === "running"
    ? [
        { key: "stop", label: "Stop", icon: "■", color: "#f85149" },
        { key: "restart", label: "Restart", icon: "↻", color: "#f0883e" },
        { key: "pause", label: "Pause", icon: "⏸", color: "#f0883e" },
        { key: "shell", label: "Shell", icon: ">_", color: "#58a6ff" },
      ]
    : container.status === "paused"
    ? [
        { key: "unpause", label: "Unpause", icon: "▸", color: "#3fb950" },
        { key: "stop", label: "Stop", icon: "■", color: "#f85149" },
      ]
    : [
        { key: "start", label: "Start", icon: "▸", color: "#3fb950" },
      ];

  const [hovered, setHovered] = useState(-1);

  return (
    <div
      style={{
        position: "absolute",
        left: 205,
        top: position,
        background: "#1c2128",
        border: "1px solid #30363d",
        borderRadius: 6,
        padding: "4px 0",
        zIndex: 50,
        minWidth: 140,
        boxShadow: "0 8px 24px rgba(0,0,0,0.4)",
      }}
    >
      <div style={{ padding: "4px 12px 6px", color: "#8b949e", fontSize: 10, textTransform: "uppercase", letterSpacing: 1 }}>
        {container.name}
      </div>
      {actions.map((a, i) => (
        <div
          key={a.key}
          onMouseEnter={() => setHovered(i)}
          onMouseLeave={() => setHovered(-1)}
          onClick={(e) => { e.stopPropagation(); onAction(a.key); onClose(); }}
          style={{
            padding: "5px 12px",
            display: "flex",
            alignItems: "center",
            gap: 8,
            cursor: "pointer",
            background: hovered === i ? "#30363d" : "transparent",
            color: hovered === i ? a.color : "#c9d1d9",
            fontSize: 12,
          }}
        >
          <span style={{ color: a.color, width: 16, textAlign: "center", fontSize: 11 }}>{a.icon}</span>
          {a.label}
        </div>
      ))}
    </div>
  );
}

export default function DockerLogMonitor() {
  const [containers, setContainers] = useState(INITIAL_CONTAINERS);
  const [selectedContainers, setSelectedContainers] = useState(new Set(INITIAL_CONTAINERS.map((c) => c.id)));
  const [showTimestamps, setShowTimestamps] = useState(true);
  const [wrapLines, setWrapLines] = useState(false);
  const [frozen, setFrozen] = useState(false);
  const [cursorLine, setCursorLine] = useState(-1);
  const [selectedLines, setSelectedLines] = useState(new Set());
  const [selectionAnchor, setSelectionAnchor] = useState(null);
  const [logs, setLogs] = useState(() => generateLogs(80));
  const [sidebarFocused, setSidebarFocused] = useState(false);
  const [sidebarCursor, setSidebarCursor] = useState(0);
  const [filterLevel, setFilterLevel] = useState("ALL");
  const [showHelp, setShowHelp] = useState(false);
  const [copied, setCopied] = useState(false);
  const [searchMode, setSearchMode] = useState(false);
  const [searchQuery, setSearchQuery] = useState("");
  const [isRegex, setIsRegex] = useState(false);
  const [regexError, setRegexError] = useState(null);
  const [shellContainer, setShellContainer] = useState(null);
  const [shellHeight, setShellHeight] = useState(200);
  const [actionMenu, setActionMenu] = useState(null); // { containerId, yPos }
  const [notification, setNotification] = useState(null);
  const [focusArea, setFocusArea] = useState("logs"); // "logs" | "sidebar" | "shell"
  const logRef = useRef(null);
  const resizing = useRef(false);

  const activeContainers = containers.filter((c) => selectedContainers.has(c.id) && c.status === "running");

  const searchRegex = isRegex && searchQuery ? tryRegex(searchQuery) : null;

  useEffect(() => {
    if (isRegex && searchQuery) {
      if (tryRegex(searchQuery)) setRegexError(null);
      else setRegexError("invalid regex");
    } else {
      setRegexError(null);
    }
  }, [searchQuery, isRegex]);

  const filteredLogs = logs.filter((l) => {
    if (!selectedContainers.has(l.container.id)) return false;
    if (filterLevel !== "ALL" && l.level !== filterLevel) return false;
    if (searchQuery) {
      if (isRegex && searchRegex) {
        if (!searchRegex.test(l.message)) return false;
      } else if (!l.message.toLowerCase().includes(searchQuery.toLowerCase())) {
        return false;
      }
    }
    return true;
  });

  // streaming new logs
  useEffect(() => {
    if (frozen) return;
    const interval = setInterval(() => {
      const running = containers.filter((c) => selectedContainers.has(c.id) && c.status === "running");
      const c = running[Math.floor(Math.random() * running.length)];
      if (!c) return;
      const templates = LOG_TEMPLATES[c.id];
      const t = templates[Math.floor(Math.random() * templates.length)];
      setLogs((prev) => [
        ...prev.slice(-200),
        { id: Date.now(), container: c, timestamp: generateTimestamp(0), level: t.level, message: t.msg },
      ]);
    }, 600);
    return () => clearInterval(interval);
  }, [frozen, selectedContainers, containers]);

  useEffect(() => {
    if (!frozen && logRef.current) logRef.current.scrollTop = logRef.current.scrollHeight;
  }, [logs, frozen]);

  const showNotif = (msg) => {
    setNotification(msg);
    setTimeout(() => setNotification(null), 2000);
  };

  const handleContainerAction = (containerId, action) => {
    setContainers((prev) =>
      prev.map((c) => {
        if (c.id !== containerId) return c;
        switch (action) {
          case "stop": showNotif(`Stopping ${c.name}...`); return { ...c, status: "stopped" };
          case "start": showNotif(`Starting ${c.name}...`); return { ...c, status: "running" };
          case "restart": showNotif(`Restarting ${c.name}...`); return { ...c, status: "running" };
          case "pause": showNotif(`Pausing ${c.name}...`); return { ...c, status: "paused" };
          case "unpause": showNotif(`Unpausing ${c.name}...`); return { ...c, status: "running" };
          case "shell": setShellContainer(c); setFocusArea("shell"); return c;
          default: return c;
        }
      })
    );
  };

  const handleCopy = useCallback(() => {
    if (selectedLines.size === 0) return;
    const sorted = [...selectedLines].sort((a, b) => a - b);
    const text = sorted
      .map((idx) => {
        const l = filteredLogs[idx];
        if (!l) return "";
        const parts = [];
        if (showTimestamps) parts.push(l.timestamp);
        parts.push(`[${l.container.name}]`);
        parts.push(l.message);
        return parts.join(" ");
      })
      .join("\n");
    navigator.clipboard?.writeText(text);
    setCopied(true);
    setTimeout(() => setCopied(false), 1500);
  }, [selectedLines, filteredLogs, showTimestamps]);

  // resize shell panel
  useEffect(() => {
    const onMove = (e) => {
      if (!resizing.current) return;
      const vh = window.innerHeight;
      const newH = vh - e.clientY - 28; // 28 for status bar
      setShellHeight(Math.max(80, Math.min(vh * 0.6, newH)));
    };
    const onUp = () => { resizing.current = false; };
    window.addEventListener("mousemove", onMove);
    window.addEventListener("mouseup", onUp);
    return () => { window.removeEventListener("mousemove", onMove); window.removeEventListener("mouseup", onUp); };
  }, []);

  useEffect(() => {
    const handler = (e) => {
      // close menus
      if (actionMenu && e.key === "Escape") {
        e.preventDefault();
        setActionMenu(null);
        return;
      }
      if (showHelp && e.key === "Escape") {
        e.preventDefault();
        setShowHelp(false);
        return;
      }

      // shell focused - let shell handle its own keys
      if (focusArea === "shell") {
        if (e.key === "Escape") {
          e.preventDefault();
          setFocusArea("logs");
          return;
        }
        // don't intercept keys when shell is focused
        return;
      }

      if (searchMode) {
        if (e.key === "Escape") {
          setSearchMode(false);
          setSearchQuery("");
        } else if (e.key === "Enter") {
          setSearchMode(false);
        } else if (e.key === "Backspace") {
          setSearchQuery((q) => q.slice(0, -1));
        } else if (e.key === "Tab") {
          e.preventDefault();
          setIsRegex((r) => !r);
        } else if (e.key.length === 1 && !e.ctrlKey && !e.metaKey) {
          setSearchQuery((q) => q + e.key);
        }
        e.preventDefault();
        return;
      }

      // global keys
      if (e.key === "?") { e.preventDefault(); setShowHelp((h) => !h); return; }
      if (e.key === "/") { e.preventDefault(); setSearchMode(true); setSearchQuery(""); return; }
      if (e.key === "t" && !e.ctrlKey) { e.preventDefault(); setShowTimestamps((s) => !s); return; }
      if (e.key === "w" && !e.ctrlKey) { e.preventDefault(); setWrapLines((w) => !w); return; }
      if (e.key === "f" && !e.ctrlKey) {
        e.preventDefault();
        setFrozen((f) => {
          if (!f) setCursorLine(filteredLogs.length - 1);
          else { setCursorLine(-1); setSelectedLines(new Set()); setSelectionAnchor(null); }
          return !f;
        });
        return;
      }
      if (e.key === "Tab") {
        e.preventDefault();
        if (shellContainer) {
          // cycle: sidebar -> logs -> shell
          setFocusArea((f) => f === "sidebar" ? "logs" : f === "logs" ? "shell" : "sidebar");
          setSidebarFocused((f) => focusArea === "logs" ? false : focusArea === "shell" ? true : false);
        } else {
          setSidebarFocused((f) => !f);
          setFocusArea((f) => f === "sidebar" ? "logs" : "sidebar");
        }
        return;
      }
      if (e.key === "l" && !e.ctrlKey) {
        e.preventDefault();
        const levels = ["ALL", "ERROR", "WARN", "INFO", "DEBUG"];
        setFilterLevel((f) => levels[(levels.indexOf(f) + 1) % levels.length]);
        return;
      }
      if (e.key === "x" && !e.ctrlKey && !e.metaKey) {
        e.preventDefault();
        if (shellContainer) { setShellContainer(null); setFocusArea("logs"); }
        return;
      }

      // sidebar keys
      if (focusArea === "sidebar") {
        if (e.key === "ArrowUp" || e.key === "k") {
          e.preventDefault();
          setSidebarCursor((c) => Math.max(0, c - 1));
          setActionMenu(null);
        } else if (e.key === "ArrowDown" || e.key === "j") {
          e.preventDefault();
          setSidebarCursor((c) => Math.min(containers.length - 1, c + 1));
          setActionMenu(null);
        } else if (e.key === " ") {
          e.preventDefault();
          const cid = containers[sidebarCursor].id;
          setSelectedContainers((prev) => {
            const next = new Set(prev);
            if (next.has(cid)) next.delete(cid);
            else next.add(cid);
            return next;
          });
        } else if (e.key === "Enter") {
          e.preventDefault();
          const c = containers[sidebarCursor];
          setActionMenu((m) => m?.containerId === c.id ? null : { containerId: c.id, yPos: 74 + sidebarCursor * 28 });
        } else if (e.key === "a" || e.key === "A") {
          e.preventDefault();
          if (selectedContainers.size === containers.length) setSelectedContainers(new Set());
          else setSelectedContainers(new Set(containers.map((c) => c.id)));
        } else if (e.key === "s" || e.key === "S") {
          // quick shell for focused container
          e.preventDefault();
          const c = containers[sidebarCursor];
          if (c.status === "running") {
            setShellContainer(c);
            setFocusArea("shell");
          }
        }
        return;
      }

      // log navigation (only when frozen)
      if (frozen) {
        if (e.key === "ArrowUp" || e.key === "k") {
          e.preventDefault();
          setCursorLine((c) => Math.max(0, c - 1));
          if (!e.shiftKey) { setSelectedLines(new Set()); setSelectionAnchor(null); }
        } else if (e.key === "ArrowDown" || e.key === "j") {
          e.preventDefault();
          setCursorLine((c) => Math.min(filteredLogs.length - 1, c + 1));
          if (!e.shiftKey) { setSelectedLines(new Set()); setSelectionAnchor(null); }
        } else if (e.key === " ") {
          e.preventDefault();
          setSelectedLines((prev) => {
            const next = new Set(prev);
            if (next.has(cursorLine)) next.delete(cursorLine);
            else next.add(cursorLine);
            return next;
          });
          setSelectionAnchor(cursorLine);
        } else if (e.key === "c" || e.key === "y") {
          e.preventDefault();
          handleCopy();
        } else if (e.key === "G") {
          e.preventDefault();
          setCursorLine(filteredLogs.length - 1);
        } else if (e.key === "g") {
          e.preventDefault();
          setCursorLine(0);
        } else if (e.key === "Escape") {
          e.preventDefault();
          setSelectedLines(new Set());
          setSelectionAnchor(null);
        }
        if (e.shiftKey && (e.key === "ArrowUp" || e.key === "ArrowDown" || e.key === "k" || e.key === "j")) {
          setCursorLine((cur) => {
            const anchor = selectionAnchor ?? cur;
            if (selectionAnchor === null) setSelectionAnchor(cur);
            const newCur = e.key === "ArrowUp" || e.key === "k" ? Math.max(0, cur - 1) : Math.min(filteredLogs.length - 1, cur + 1);
            const start = Math.min(anchor, newCur);
            const end = Math.max(anchor, newCur);
            const ns = new Set();
            for (let i = start; i <= end; i++) ns.add(i);
            setSelectedLines(ns);
            return newCur;
          });
        }
        if (e.key === "PageUp") { e.preventDefault(); setCursorLine((c) => Math.max(0, c - 20)); }
        if (e.key === "PageDown") { e.preventDefault(); setCursorLine((c) => Math.min(filteredLogs.length - 1, c + 20)); }
      }
    };
    window.addEventListener("keydown", handler);
    return () => window.removeEventListener("keydown", handler);
  }, [frozen, cursorLine, filteredLogs, focusArea, sidebarCursor, containers, selectedContainers, selectionAnchor, showHelp, searchMode, handleCopy, actionMenu, shellContainer]);

  useEffect(() => {
    if (frozen && logRef.current && cursorLine >= 0) {
      const el = logRef.current.children[cursorLine];
      if (el) el.scrollIntoView({ block: "nearest" });
    }
  }, [cursorLine, frozen]);

  const levelBadge = (level) => (
    <span style={{ color: LEVEL_COLORS[level], fontWeight: level === "ERROR" ? 700 : 400 }}>
      {level.padEnd(5)}
    </span>
  );

  return (
    <div
      tabIndex={0}
      style={{
        fontFamily: "'JetBrains Mono', 'Fira Code', 'Cascadia Code', monospace",
        fontSize: 13,
        background: "#0d1117",
        color: "#c9d1d9",
        height: "100vh",
        display: "flex",
        flexDirection: "column",
        outline: "none",
        overflow: "hidden",
      }}
      onClick={() => { setActionMenu(null); }}
    >
      {/* Title bar */}
      <div style={{ background: "#161b22", borderBottom: "1px solid #30363d", padding: "6px 12px", display: "flex", alignItems: "center", justifyContent: "space-between", flexShrink: 0 }}>
        <div style={{ display: "flex", alignItems: "center", gap: 10 }}>
          <span style={{ color: "#58a6ff", fontWeight: 700 }}>◉ dklog</span>
          <span style={{ color: "#484f58" }}>│</span>
          <span style={{ color: "#8b949e" }}>project: <span style={{ color: "#c9d1d9" }}>myapp</span></span>
          <span style={{ color: "#484f58" }}>│</span>
          <span style={{ color: "#8b949e" }}>containers: <span style={{ color: "#c9d1d9" }}>{selectedContainers.size}/{containers.length}</span></span>
        </div>
        <div style={{ display: "flex", alignItems: "center", gap: 10 }}>
          {notification && <span style={{ color: "#f0883e", fontSize: 12 }}>{notification}</span>}
          {copied && <span style={{ color: "#3fb950", fontSize: 12 }}>✓ copied</span>}
          {searchMode && (
            <span style={{ color: "#f0883e", fontSize: 12 }}>
              {isRegex ? "regex:" : "/"}{searchQuery}
              <span style={{ animation: "blink 1s step-end infinite" }}>▌</span>
              {regexError && <span style={{ color: "#f85149", marginLeft: 6 }}>{regexError}</span>}
              <span style={{ color: "#484f58", marginLeft: 6, fontSize: 10 }}>Tab: {isRegex ? "regex" : "text"}</span>
            </span>
          )}
          {searchQuery && !searchMode && (
            <span style={{ color: "#8b949e", fontSize: 12 }}>
              {isRegex ? "regex" : "search"}: {searchQuery}
              {filteredLogs.length > 0 && <span style={{ color: "#484f58" }}> ({filteredLogs.length} matches)</span>}
            </span>
          )}
          {filterLevel !== "ALL" && (
            <span style={{ background: LEVEL_COLORS[filterLevel] + "22", color: LEVEL_COLORS[filterLevel], padding: "1px 6px", borderRadius: 3, fontSize: 11 }}>
              {filterLevel}
            </span>
          )}
          {frozen && <span style={{ background: "#f0883e22", color: "#f0883e", padding: "1px 6px", borderRadius: 3, fontSize: 11, fontWeight: 600 }}>❄ FROZEN</span>}
          {wrapLines && <span style={{ color: "#484f58", fontSize: 11 }}>wrap:on</span>}
          {showTimestamps && <span style={{ color: "#484f58", fontSize: 11 }}>ts:on</span>}
          <span style={{ color: "#484f58", fontSize: 11 }}>? help</span>
        </div>
      </div>

      {/* Main area */}
      <div style={{ display: "flex", flex: 1, overflow: "hidden", position: "relative" }}>
        {/* Sidebar */}
        <div style={{ width: 200, background: "#0d1117", borderRight: "1px solid #30363d", display: "flex", flexDirection: "column", flexShrink: 0, position: "relative" }}>
          <div style={{ padding: "6px 10px", color: focusArea === "sidebar" ? "#58a6ff" : "#8b949e", fontSize: 11, textTransform: "uppercase", letterSpacing: 1, borderBottom: "1px solid #30363d", background: focusArea === "sidebar" ? "#161b22" : "transparent" }}>
            Containers {focusArea === "sidebar" && "▸"}
          </div>
          {containers.map((c, i) => {
            const active = selectedContainers.has(c.id);
            const focused = focusArea === "sidebar" && sidebarCursor === i;
            const si = STATUS_ICONS[c.status] || STATUS_ICONS.stopped;
            return (
              <div
                key={c.id}
                style={{
                  padding: "4px 10px",
                  display: "flex",
                  alignItems: "center",
                  gap: 8,
                  background: focused ? "#1f2937" : "transparent",
                  borderLeft: focused ? "2px solid #58a6ff" : "2px solid transparent",
                  opacity: active ? 1 : 0.4,
                  cursor: "pointer",
                  position: "relative",
                }}
                onClick={(e) => { e.stopPropagation(); }}
              >
                <span style={{ color: active ? "#3fb950" : "#484f58", fontSize: 10 }}>{active ? "●" : "○"}</span>
                <span style={{ color: c.color, fontWeight: 600, fontSize: 12, flex: 1 }}>{c.name}</span>
                <span style={{ color: si.color, fontSize: 9, marginRight: 2 }} title={c.status}>{si.icon}</span>
                {shellContainer?.id === c.id && <span style={{ color: "#58a6ff", fontSize: 9 }} title="shell open">{">_"}</span>}
              </div>
            );
          })}
          <div style={{ flex: 1 }} />
          <div style={{ padding: "6px 10px", borderTop: "1px solid #30363d", fontSize: 11, color: "#484f58" }}>
            <div>⇥ Tab focus</div>
            <div>⎵ toggle log</div>
            <div>↵ actions</div>
            <div>s shell</div>
            <div>a select all</div>
          </div>

          {/* Action menu */}
          {actionMenu && (
            <ActionMenu
              container={containers.find((c) => c.id === actionMenu.containerId)}
              position={actionMenu.yPos}
              onAction={(action) => handleContainerAction(actionMenu.containerId, action)}
              onClose={() => setActionMenu(null)}
            />
          )}
        </div>

        {/* Right panel: logs + shell */}
        <div style={{ flex: 1, display: "flex", flexDirection: "column", overflow: "hidden" }}>
          {/* Log area */}
          <div style={{ flex: 1, display: "flex", flexDirection: "column", overflow: "hidden", minHeight: 100 }}>
            <div
              ref={logRef}
              style={{ flex: 1, overflow: "auto", padding: "2px 0" }}
            >
              {filteredLogs.map((log, idx) => {
                const isCursor = frozen && cursorLine === idx;
                const isSelected = selectedLines.has(idx);
                return (
                  <div
                    key={log.id + "-" + idx}
                    style={{
                      padding: "1px 12px",
                      display: "flex",
                      gap: 0,
                      whiteSpace: wrapLines ? "pre-wrap" : "nowrap",
                      wordBreak: wrapLines ? "break-all" : "normal",
                      background: isSelected ? "#1f3a5f" : isCursor ? "#1c2333" : "transparent",
                      borderLeft: isCursor ? "2px solid #58a6ff" : isSelected ? "2px solid #1f6feb" : "2px solid transparent",
                      lineHeight: "20px",
                    }}
                  >
                    {frozen && (
                      <span style={{ color: "#30363d", width: 40, textAlign: "right", marginRight: 8, flexShrink: 0, userSelect: "none" }}>
                        {idx + 1}
                      </span>
                    )}
                    {showTimestamps && (
                      <span style={{ color: "#484f58", marginRight: 8, flexShrink: 0 }}>{log.timestamp}</span>
                    )}
                    <span style={{ color: log.container.color, marginRight: 8, flexShrink: 0, fontWeight: 600, width: 80 }}>
                      {log.container.name}
                    </span>
                    <span style={{ marginRight: 8, flexShrink: 0 }}>{levelBadge(log.level)}</span>
                    <span
                      style={{
                        color: log.level === "ERROR" ? "#f85149" : log.level === "WARN" ? "#d29922" : "#c9d1d9",
                        overflow: wrapLines ? "visible" : "hidden",
                        textOverflow: wrapLines ? "unset" : "ellipsis",
                      }}
                    >
                      {searchQuery && (isRegex ? searchRegex : searchQuery)
                        ? highlightMatches(log.message, isRegex ? searchRegex : new RegExp(searchQuery.replace(/[.*+?^${}()|[\]\\]/g, "\\$&"), "i"))
                        : log.message}
                    </span>
                  </div>
                );
              })}
            </div>
          </div>

          {/* Shell panel */}
          {shellContainer && (
            <>
              {/* Resize handle */}
              <div
                onMouseDown={() => { resizing.current = true; }}
                style={{
                  height: 4,
                  background: focusArea === "shell" ? "#58a6ff" : "#30363d",
                  cursor: "row-resize",
                  flexShrink: 0,
                }}
              />
              {/* Shell tab bar */}
              <div style={{
                background: "#161b22",
                borderBottom: "1px solid #30363d",
                padding: "0 12px",
                display: "flex",
                alignItems: "center",
                height: 28,
                flexShrink: 0,
              }}>
                <div style={{
                  display: "flex",
                  alignItems: "center",
                  gap: 6,
                  padding: "0 10px",
                  height: "100%",
                  borderBottom: "2px solid #58a6ff",
                  color: "#c9d1d9",
                  fontSize: 12,
                }}>
                  <span style={{ color: "#58a6ff" }}>{">_"}</span>
                  <span style={{ color: shellContainer.color }}>{shellContainer.name}</span>
                  <span style={{ color: "#484f58" }}>shell</span>
                </div>
                <div style={{ flex: 1 }} />
                <span
                  onClick={() => { setShellContainer(null); setFocusArea("logs"); }}
                  style={{ color: "#484f58", cursor: "pointer", fontSize: 11, padding: "0 4px" }}
                  title="Close shell (x)"
                >
                  ✕
                </span>
              </div>
              {/* Shell content */}
              <div
                style={{ height: shellHeight, flexShrink: 0, borderTop: "1px solid #21262d" }}
                onClick={() => setFocusArea("shell")}
              >
                <ShellPanel container={shellContainer} onClose={() => { setShellContainer(null); setFocusArea("logs"); }} />
              </div>
            </>
          )}
        </div>
      </div>

      {/* Status bar */}
      <div style={{ background: frozen ? "#1c1e2a" : "#161b22", borderTop: "1px solid #30363d", padding: "4px 12px", display: "flex", justifyContent: "space-between", fontSize: 12, flexShrink: 0 }}>
        <div style={{ display: "flex", gap: 16 }}>
          <span style={{ color: "#8b949e" }}><span style={{ color: "#58a6ff" }}>f</span> freeze</span>
          <span style={{ color: "#8b949e" }}><span style={{ color: "#58a6ff" }}>t</span> timestamps</span>
          <span style={{ color: "#8b949e" }}><span style={{ color: "#58a6ff" }}>w</span> wrap{wrapLines ? ":on" : ""}</span>
          <span style={{ color: "#8b949e" }}><span style={{ color: "#58a6ff" }}>l</span> {filterLevel.toLowerCase()}</span>
          <span style={{ color: "#8b949e" }}><span style={{ color: "#58a6ff" }}>/</span> search</span>
          {shellContainer && <span style={{ color: "#8b949e" }}><span style={{ color: "#58a6ff" }}>x</span> close shell</span>}
          {frozen && (
            <>
              <span style={{ color: "#484f58" }}>│</span>
              <span style={{ color: "#8b949e" }}><span style={{ color: "#58a6ff" }}>⎵</span> select</span>
              <span style={{ color: "#8b949e" }}><span style={{ color: "#58a6ff" }}>⇧↑↓</span> range</span>
              <span style={{ color: "#8b949e" }}><span style={{ color: "#58a6ff" }}>y</span> copy</span>
            </>
          )}
        </div>
        <div style={{ display: "flex", gap: 12 }}>
          {focusArea === "shell" && <span style={{ color: "#58a6ff", fontSize: 11 }}>SHELL</span>}
          {selectedLines.size > 0 && <span style={{ color: "#f0883e" }}>{selectedLines.size} selected</span>}
          <span style={{ color: "#484f58" }}>{filteredLogs.length} lines</span>
          {frozen && cursorLine >= 0 && <span style={{ color: "#484f58" }}>ln {cursorLine + 1}</span>}
        </div>
      </div>

      {/* Help overlay */}
      {showHelp && (
        <div style={{ position: "fixed", inset: 0, background: "rgba(0,0,0,0.7)", display: "flex", alignItems: "center", justifyContent: "center", zIndex: 100 }}>
          <div style={{ background: "#161b22", border: "1px solid #30363d", borderRadius: 8, padding: "20px 28px", maxWidth: 560, width: "90%", maxHeight: "80vh", overflowY: "auto" }}>
            <div style={{ color: "#58a6ff", fontWeight: 700, marginBottom: 16, fontSize: 15 }}>Keyboard Shortcuts</div>
            {[
              ["General", [
                ["?", "Toggle this help"],
                ["f", "Freeze / unfreeze logs"],
                ["t", "Toggle timestamps"],
                ["w", "Toggle line wrap"],
                ["l", "Cycle log levels (ALL → ERROR → WARN → INFO → DEBUG)"],
                ["/", "Search logs (Tab to toggle regex mode)"],
                ["x", "Close shell panel"],
                ["Tab", "Cycle focus: sidebar → logs → shell"],
              ]],
              ["Sidebar (when focused)", [
                ["↑/↓ or j/k", "Navigate containers"],
                ["Space", "Toggle container log on/off"],
                ["Enter", "Open actions menu (start/stop/restart/pause)"],
                ["s", "Open shell for focused container"],
                ["a", "Select all / deselect all"],
              ]],
              ["Logs (when frozen)", [
                ["↑/↓ or j/k", "Move cursor"],
                ["g / G", "Jump to top / bottom"],
                ["PgUp/PgDn", "Page up / down"],
                ["Space", "Toggle select line"],
                ["Shift+↑/↓", "Range select"],
                ["y or c", "Copy selected lines"],
                ["Esc", "Clear selection"],
              ]],
              ["Shell", [
                ["↑/↓", "Command history"],
                ["Enter", "Execute command"],
                ["Esc", "Return focus to logs"],
              ]],
            ].map(([section, keys]) => (
              <div key={section} style={{ marginBottom: 14 }}>
                <div style={{ color: "#8b949e", fontSize: 11, textTransform: "uppercase", letterSpacing: 1, marginBottom: 6 }}>{section}</div>
                {keys.map(([key, desc]) => (
                  <div key={key} style={{ display: "flex", gap: 12, padding: "2px 0" }}>
                    <span style={{ color: "#58a6ff", width: 110, textAlign: "right", flexShrink: 0 }}>{key}</span>
                    <span style={{ color: "#c9d1d9" }}>{desc}</span>
                  </div>
                ))}
              </div>
            ))}
            <div style={{ color: "#484f58", fontSize: 11, marginTop: 12, textAlign: "center" }}>Press ? or Esc to close</div>
          </div>
        </div>
      )}

      <style>{`
        @keyframes blink { 50% { opacity: 0; } }
        ::-webkit-scrollbar { width: 6px; }
        ::-webkit-scrollbar-track { background: #0d1117; }
        ::-webkit-scrollbar-thumb { background: #30363d; border-radius: 3px; }
        ::-webkit-scrollbar-thumb:hover { background: #484f58; }
        * { box-sizing: border-box; }
      `}</style>
    </div>
  );
}
