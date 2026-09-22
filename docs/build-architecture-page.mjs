import { readFile, writeFile } from "node:fs/promises";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const docsDir = dirname(fileURLToPath(import.meta.url));
const sourceName = process.argv[2] ?? "architecture-design.md";
const outputName = process.argv[3] ?? sourceName.replace(/\.md$/, ".html");
const markdownPath = join(docsDir, sourceName);
const outputPath = join(docsDir, outputName);
const markdown = await readFile(markdownPath, "utf8");
const embeddedMarkdown = JSON.stringify(markdown).replaceAll("<", "\\u003c");
const isBackendGuide = sourceName === "backend-detailed-design.md";
const isPWAGuide = sourceName === "pwa-detailed-design.md";
const page = isBackendGuide ? {
  description: "As-built Pinerary backend design and repository guide",
  browserTitle: "Pinerary — Backend Detailed Design",
  eyebrow: "As-built system · Repository guide",
  heroTitle: "Backend Detailed<br />Design",
  copy: "A code-level guide to the Go API, PostGIS model, durable jobs, route processing, private media, road search, and public itinerary flows.",
  status: "As-built backend guide",
  footer: "Pinerary backend design artifact",
} : isPWAGuide ? {
  description: "As-built Pinerary PWA design and repository guide",
  browserTitle: "Pinerary — PWA Detailed Design",
  eyebrow: "As-built client · Repository guide",
  heroTitle: "PWA Detailed<br />Design",
  copy: "A code-level guide to the static Next.js client, IndexedDB outbox, foreground tracking, maps, offline photos, road search, and itinerary sharing.",
  status: "PWA MVP implementation guide",
  footer: "Pinerary PWA design artifact",
} : {
  description: "Pinerary architecture and detailed technical design",
  browserTitle: "Pinerary — Architecture & Detailed Design",
  eyebrow: "System design · Architecture review",
  heroTitle: "Architecture &<br />Detailed Design",
  copy: "A backend-led, offline-first travel journal with automatic route tracking, geospatial discovery, private media, and shareable itineraries.",
  status: "Backend + PWA implemented · Hardening next",
  footer: "Pinerary architecture artifact",
};

const html = String.raw`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <meta name="color-scheme" content="light dark" />
  <meta name="description" content="${page.description}" />
  <title>${page.browserTitle}</title>
  <style>
    :root {
      --bg: #f4f7f5;
      --surface: rgba(255, 255, 255, 0.9);
      --surface-solid: #ffffff;
      --surface-muted: #eef4f0;
      --text: #17211c;
      --muted: #627169;
      --line: #dce6df;
      --accent: #087f5b;
      --accent-strong: #056347;
      --accent-soft: #dff5eb;
      --blue: #315caa;
      --code: #13251d;
      --code-text: #dff8ea;
      --shadow: 0 20px 60px rgba(26, 58, 43, 0.08);
      --radius: 18px;
      --content-width: 920px;
    }

    :root[data-theme="dark"] {
      --bg: #0c1410;
      --surface: rgba(20, 31, 25, 0.92);
      --surface-solid: #141f19;
      --surface-muted: #1a2a21;
      --text: #e9f3ed;
      --muted: #9db0a5;
      --line: #293a31;
      --accent: #4bd3a2;
      --accent-strong: #7ae5bd;
      --accent-soft: #173d2f;
      --blue: #8bb3ff;
      --code: #08110d;
      --code-text: #dff8ea;
      --shadow: 0 24px 70px rgba(0, 0, 0, 0.28);
    }

    * { box-sizing: border-box; }

    html { scroll-behavior: smooth; }

    body {
      margin: 0;
      color: var(--text);
      background:
        radial-gradient(circle at 10% 0%, rgba(75, 211, 162, 0.12), transparent 28rem),
        radial-gradient(circle at 92% 8%, rgba(49, 92, 170, 0.1), transparent 26rem),
        var(--bg);
      font: 16px/1.72 ui-sans-serif, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
      -webkit-font-smoothing: antialiased;
    }

    a { color: var(--accent); }

    button, input { font: inherit; }

    .progress {
      position: fixed;
      z-index: 100;
      top: 0;
      left: 0;
      width: 0;
      height: 3px;
      background: linear-gradient(90deg, var(--accent), #47b8dc);
    }

    .topbar {
      position: sticky;
      z-index: 50;
      top: 0;
      display: flex;
      align-items: center;
      justify-content: space-between;
      min-height: 64px;
      padding: 0 28px;
      border-bottom: 1px solid color-mix(in srgb, var(--line) 78%, transparent);
      background: color-mix(in srgb, var(--bg) 84%, transparent);
      backdrop-filter: blur(18px);
    }

    .brand {
      display: flex;
      align-items: center;
      gap: 11px;
      font-weight: 760;
      letter-spacing: -0.02em;
    }

    .brand-mark {
      display: grid;
      width: 34px;
      height: 34px;
      place-items: center;
      border-radius: 11px;
      color: white;
      background: linear-gradient(145deg, #0a8c64, #315caa);
      box-shadow: 0 8px 24px rgba(8, 127, 91, 0.24);
    }

    .top-actions { display: flex; gap: 8px; }

    .icon-button {
      min-height: 38px;
      padding: 7px 12px;
      border: 1px solid var(--line);
      border-radius: 10px;
      color: var(--text);
      background: var(--surface);
      cursor: pointer;
    }

    .icon-button:hover { border-color: var(--accent); }

    .hero {
      max-width: 1240px;
      margin: 0 auto;
      padding: 72px 30px 44px;
    }

    .eyebrow {
      margin: 0 0 12px;
      color: var(--accent);
      font-size: 0.78rem;
      font-weight: 800;
      letter-spacing: 0.13em;
      text-transform: uppercase;
    }

    .hero h1 {
      max-width: 820px;
      margin: 0;
      font-size: clamp(2.5rem, 6vw, 5.4rem);
      line-height: 0.98;
      letter-spacing: -0.065em;
    }

    .hero-copy {
      max-width: 720px;
      margin: 24px 0 0;
      color: var(--muted);
      font-size: clamp(1.05rem, 2vw, 1.25rem);
    }

    .chips {
      display: flex;
      flex-wrap: wrap;
      gap: 10px;
      margin-top: 28px;
    }

    .chip {
      padding: 7px 12px;
      border: 1px solid var(--line);
      border-radius: 999px;
      color: var(--muted);
      background: var(--surface);
      font-size: 0.85rem;
      font-weight: 650;
    }

    .chip.primary {
      border-color: color-mix(in srgb, var(--accent) 32%, var(--line));
      color: var(--accent-strong);
      background: var(--accent-soft);
    }

    .layout {
      display: grid;
      grid-template-columns: 270px minmax(0, var(--content-width));
      gap: 44px;
      justify-content: center;
      max-width: 1280px;
      margin: 0 auto;
      padding: 0 28px 100px;
    }

    .sidebar {
      position: sticky;
      top: 88px;
      align-self: start;
      max-height: calc(100vh - 110px);
      overflow: auto;
      padding: 18px;
      border: 1px solid var(--line);
      border-radius: var(--radius);
      background: var(--surface);
      box-shadow: var(--shadow);
      backdrop-filter: blur(18px);
    }

    .search {
      width: 100%;
      padding: 10px 12px;
      border: 1px solid var(--line);
      border-radius: 10px;
      outline: none;
      color: var(--text);
      background: var(--surface-muted);
    }

    .search:focus { border-color: var(--accent); }

    .toc-title {
      margin: 18px 8px 8px;
      color: var(--muted);
      font-size: 0.73rem;
      font-weight: 800;
      letter-spacing: 0.12em;
      text-transform: uppercase;
    }

    .toc { display: grid; gap: 2px; }

    .toc a {
      padding: 7px 9px;
      border-radius: 8px;
      color: var(--muted);
      font-size: 0.84rem;
      line-height: 1.35;
      text-decoration: none;
    }

    .toc a.sub { padding-left: 20px; font-size: 0.79rem; }
    .toc a:hover, .toc a.active { color: var(--accent-strong); background: var(--accent-soft); }
    .toc a[hidden] { display: none; }

    .document {
      min-width: 0;
      padding: clamp(24px, 5vw, 58px);
      border: 1px solid var(--line);
      border-radius: 24px;
      background: var(--surface-solid);
      box-shadow: var(--shadow);
    }

    .document > h1:first-child { display: none; }

    .document h2,
    .document h3,
    .document h4 {
      position: relative;
      color: var(--text);
      line-height: 1.25;
      letter-spacing: -0.035em;
      scroll-margin-top: 90px;
    }

    .document h2 {
      margin: 70px 0 20px;
      padding-top: 12px;
      border-top: 1px solid var(--line);
      font-size: clamp(1.65rem, 3vw, 2.25rem);
    }

    .document h2:first-of-type { margin-top: 20px; }
    .document h3 { margin: 42px 0 14px; font-size: 1.38rem; }
    .document h4 { margin: 28px 0 10px; font-size: 1.08rem; }

    .heading-anchor {
      position: absolute;
      left: -1.25em;
      opacity: 0;
      color: var(--muted);
      text-decoration: none;
    }

    h2:hover .heading-anchor,
    h3:hover .heading-anchor { opacity: 1; }

    .document p { margin: 0 0 18px; }
    .document strong { color: var(--text); }
    .document ul, .document ol { padding-left: 1.4rem; }
    .document li { margin: 7px 0; padding-left: 4px; }
    .document li::marker { color: var(--accent); font-weight: 800; }

    .table-wrap {
      margin: 24px 0 30px;
      overflow-x: auto;
      border: 1px solid var(--line);
      border-radius: 14px;
    }

    table {
      width: 100%;
      border-collapse: collapse;
      font-size: 0.9rem;
    }

    th, td {
      padding: 12px 14px;
      border-bottom: 1px solid var(--line);
      text-align: left;
      vertical-align: top;
    }

    th {
      color: var(--accent-strong);
      background: var(--surface-muted);
      font-size: 0.78rem;
      letter-spacing: 0.04em;
      text-transform: uppercase;
    }

    tr:last-child td { border-bottom: 0; }
    tbody tr:hover { background: color-mix(in srgb, var(--accent-soft) 45%, transparent); }

    code {
      padding: 0.14em 0.4em;
      border: 1px solid var(--line);
      border-radius: 6px;
      color: var(--accent-strong);
      background: var(--surface-muted);
      font: 0.88em/1.5 ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
    }

    pre {
      margin: 24px 0;
      overflow: auto;
      padding: 20px;
      border-radius: 14px;
      color: var(--code-text);
      background: var(--code);
      box-shadow: inset 0 0 0 1px rgba(255,255,255,0.06);
    }

    pre code {
      padding: 0;
      border: 0;
      color: inherit;
      background: transparent;
    }

    pre.mermaid {
      display: flex;
      justify-content: center;
      color: var(--text);
      background: var(--surface-muted);
    }

    pre.mermaid svg { max-width: 100%; height: auto; }

    .diagram-note {
      margin: -14px 0 24px;
      color: var(--muted);
      font-size: 0.78rem;
      text-align: center;
    }

    .reading-footer {
      max-width: var(--content-width);
      margin: 36px auto 0;
      padding: 24px;
      border-radius: 16px;
      color: var(--muted);
      background: var(--surface-muted);
      text-align: center;
    }

    .mobile-toc { display: none; }

    @media (max-width: 960px) {
      .layout { display: block; padding-inline: 16px; }
      .sidebar { display: none; }
      .hero { padding: 52px 22px 30px; }
      .document { padding: 26px 22px; border-radius: 18px; }
      .topbar { padding: 0 16px; }
      .print-label { display: none; }
      .mobile-toc {
        display: block;
        width: 100%;
        margin-bottom: 14px;
        padding: 11px 12px;
        border: 1px solid var(--line);
        border-radius: 10px;
        color: var(--text);
        background: var(--surface-solid);
      }
    }

    @media print {
      :root { --bg: white; --surface-solid: white; --text: #111; --muted: #444; --line: #ddd; }
      .topbar, .sidebar, .mobile-toc, .progress, .heading-anchor { display: none !important; }
      .hero { padding: 20px 0 30px; }
      .layout { display: block; max-width: none; padding: 0; }
      .document { max-width: none; padding: 0; border: 0; box-shadow: none; }
      .document h2 { break-after: avoid; }
      pre, table { break-inside: avoid; }
      a { color: inherit; text-decoration: none; }
    }
  </style>
</head>
<body>
  <div class="progress" id="progress"></div>
  <header class="topbar">
    <div class="brand"><span class="brand-mark">P</span><span>Pinerary</span></div>
    <div class="top-actions">
      <button class="icon-button" id="theme" type="button" aria-label="Toggle theme">◐</button>
      <button class="icon-button" type="button" onclick="window.print()">↗ <span class="print-label">Print / PDF</span></button>
    </div>
  </header>

  <section class="hero">
    <p class="eyebrow">${page.eyebrow}</p>
    <h1>${page.heroTitle}</h1>
    <p class="hero-copy">${page.copy}</p>
    <div class="chips">
      <span class="chip primary">${page.status}</span>
      <span class="chip">Go + Gin</span>
      <span class="chip">PostgreSQL + PostGIS</span>
      <span class="chip">Next.js PWA</span>
      <span class="chip">Capacitor Android · final phase</span>
      <span class="chip">Updated 21 Sep 2026</span>
    </div>
  </section>

  <main class="layout">
    <aside class="sidebar">
      <input class="search" id="search" type="search" placeholder="Filter sections…" aria-label="Filter sections" />
      <div class="toc-title">On this page</div>
      <nav class="toc" id="toc"></nav>
    </aside>

    <section>
      <select class="mobile-toc" id="mobile-toc" aria-label="Jump to section">
        <option value="">Jump to section…</option>
      </select>
      <article class="document" id="document"></article>
      <footer class="reading-footer">${page.footer} · Generated from <code>${sourceName}</code></footer>
    </section>
  </main>

  <script>
    const markdownSource = ${embeddedMarkdown};

    const escapeHTML = (value) => value
      .replaceAll("&", "&amp;")
      .replaceAll("<", "&lt;")
      .replaceAll(">", "&gt;")
      .replaceAll('"', "&quot;");

    function renderInline(source) {
      const code = [];
      let text = source.replace(/\x60([^\x60]+)\x60/g, (_, value) => {
        const key = "@@CODE" + code.length + "@@";
        code.push("<code>" + escapeHTML(value) + "</code>");
        return key;
      });
      text = escapeHTML(text);
      text = text.replace(/\[([^\]]+)\]\((https?:\/\/[^)]+)\)/g, '<a href="$2" target="_blank" rel="noreferrer">$1</a>');
      text = text.replace(/\*\*([^*]+)\*\*/g, "<strong>$1</strong>");
      code.forEach((value, index) => { text = text.replace("@@CODE" + index + "@@", value); });
      return text;
    }

    function cells(line) {
      return line.trim().replace(/^\|/, "").replace(/\|$/, "").split("|").map((cell) => cell.trim());
    }

    function isTableDivider(line) {
      const values = cells(line);
      return values.length > 0 && values.every((value) => /^:?-{3,}:?$/.test(value));
    }

    function renderMarkdown(source) {
      const lines = source.replaceAll("\r\n", "\n").split("\n");
      const output = [];
      let i = 0;

      while (i < lines.length) {
        const line = lines[i];
        if (!line.trim()) { i += 1; continue; }

        const fence = line.match(/^\x60\x60\x60([^\s]*)\s*$/);
        if (fence) {
          const language = fence[1];
          const body = [];
          i += 1;
          while (i < lines.length && !/^\x60\x60\x60/.test(lines[i])) body.push(lines[i++]);
          i += 1;
          if (language === "mermaid") {
            output.push('<pre class="mermaid">' + escapeHTML(body.join("\n")) + "</pre>");
            output.push('<div class="diagram-note">Interactive diagram · text fallback remains available offline</div>');
          } else {
            output.push("<pre><code>" + escapeHTML(body.join("\n")) + "</code></pre>");
          }
          continue;
        }

        const heading = line.match(/^(#{1,6})\s+(.+)$/);
        if (heading) {
          const level = heading[1].length;
          output.push("<h" + level + ">" + renderInline(heading[2]) + "</h" + level + ">");
          i += 1;
          continue;
        }

        if (line.trim().startsWith("|") && i + 1 < lines.length && isTableDivider(lines[i + 1])) {
          const headers = cells(line);
          const rows = [];
          i += 2;
          while (i < lines.length && lines[i].trim().startsWith("|")) rows.push(cells(lines[i++]));
          output.push('<div class="table-wrap"><table><thead><tr>' + headers.map((value) => "<th>" + renderInline(value) + "</th>").join("") + "</tr></thead><tbody>" + rows.map((row) => "<tr>" + row.map((value) => "<td>" + renderInline(value) + "</td>").join("") + "</tr>").join("") + "</tbody></table></div>");
          continue;
        }

        const unordered = line.match(/^\s*-\s+(.+)$/);
        const ordered = line.match(/^\s*\d+\.\s+(.+)$/);
        if (unordered || ordered) {
          const tag = unordered ? "ul" : "ol";
          const pattern = unordered ? /^\s*-\s+(.+)$/ : /^\s*\d+\.\s+(.+)$/;
          const items = [];
          while (i < lines.length) {
            const item = lines[i].match(pattern);
            if (!item) break;
            items.push("<li>" + renderInline(item[1]) + "</li>");
            i += 1;
          }
          output.push("<" + tag + ">" + items.join("") + "</" + tag + ">");
          continue;
        }

        const paragraph = [line.trim()];
        i += 1;
        while (i < lines.length && lines[i].trim() && !/^(#{1,6})\s/.test(lines[i]) && !/^\x60\x60\x60/.test(lines[i]) && !/^\s*(-|\d+\.)\s+/.test(lines[i]) && !(lines[i].trim().startsWith("|") && i + 1 < lines.length && isTableDivider(lines[i + 1]))) {
          paragraph.push(lines[i].trim());
          i += 1;
        }
        output.push("<p>" + renderInline(paragraph.join(" ")) + "</p>");
      }
      return output.join("\n");
    }

    const slugCounts = new Map();
    function slugify(value) {
      const base = value.toLowerCase().replace(/[^a-z0-9]+/g, "-").replace(/^-|-$/g, "") || "section";
      const count = slugCounts.get(base) || 0;
      slugCounts.set(base, count + 1);
      return count ? base + "-" + (count + 1) : base;
    }

    const documentElement = document.getElementById("document");
    documentElement.innerHTML = renderMarkdown(markdownSource);

    const toc = document.getElementById("toc");
    const mobileToc = document.getElementById("mobile-toc");
    const headings = [...documentElement.querySelectorAll("h2, h3")];
    headings.forEach((heading) => {
      heading.id = slugify(heading.textContent);
      const anchor = document.createElement("a");
      anchor.className = "heading-anchor";
      anchor.href = "#" + heading.id;
      anchor.textContent = "#";
      anchor.setAttribute("aria-label", "Link to " + heading.textContent);
      heading.prepend(anchor);

      const link = document.createElement("a");
      link.href = "#" + heading.id;
      link.textContent = heading.textContent.replace(/^#/, "").trim();
      link.dataset.target = heading.id;
      if (heading.tagName === "H3") link.className = "sub";
      toc.append(link);

      if (heading.tagName === "H2") {
        const option = document.createElement("option");
        option.value = heading.id;
        option.textContent = heading.textContent.replace(/^#/, "").trim();
        mobileToc.append(option);
      }
    });

    mobileToc.addEventListener("change", (event) => {
      if (event.target.value) document.getElementById(event.target.value)?.scrollIntoView();
      event.target.value = "";
    });

    document.getElementById("search").addEventListener("input", (event) => {
      const query = event.target.value.trim().toLowerCase();
      toc.querySelectorAll("a").forEach((link) => { link.hidden = query && !link.textContent.toLowerCase().includes(query); });
    });

    const observer = new IntersectionObserver((entries) => {
      const visible = entries.filter((entry) => entry.isIntersecting).sort((a, b) => a.boundingClientRect.top - b.boundingClientRect.top)[0];
      if (!visible) return;
      toc.querySelectorAll("a").forEach((link) => link.classList.toggle("active", link.dataset.target === visible.target.id));
    }, { rootMargin: "-15% 0px -72% 0px" });
    headings.filter((heading) => heading.tagName === "H2").forEach((heading) => observer.observe(heading));

    const progress = document.getElementById("progress");
    window.addEventListener("scroll", () => {
      const distance = document.documentElement.scrollHeight - innerHeight;
      progress.style.width = (distance > 0 ? (scrollY / distance) * 100 : 0) + "%";
    }, { passive: true });

    const themeButton = document.getElementById("theme");
    let savedTheme = null;
    try { savedTheme = localStorage.getItem("pinerary-doc-theme"); } catch (_) {}
    if (savedTheme) document.documentElement.dataset.theme = savedTheme;
    themeButton.addEventListener("click", () => {
      const next = document.documentElement.dataset.theme === "dark" ? "light" : "dark";
      document.documentElement.dataset.theme = next;
      try { localStorage.setItem("pinerary-doc-theme", next); } catch (_) {}
    });

    import("https://cdn.jsdelivr.net/npm/mermaid@11/dist/mermaid.esm.min.mjs")
      .then(({ default: mermaid }) => {
        mermaid.initialize({ startOnLoad: false, theme: document.documentElement.dataset.theme === "dark" ? "dark" : "neutral", securityLevel: "strict" });
        return mermaid.run({ nodes: document.querySelectorAll(".mermaid") });
      })
      .catch(() => {});
  </script>
</body>
</html>`;

await writeFile(outputPath, html, "utf8");
console.log(`Generated ${outputPath}`);
