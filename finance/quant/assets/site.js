(function () {
  "use strict";

  const body = document.body;
  const assetBase = body.dataset.assetBase || ".";
  const slug = body.dataset.slug || "";
  const isIndex = body.dataset.page === "index";

  const OUTCOME_COLORS = {
    matches: "#2f6f4e",
    mismatches: "#a15c16",
    unsupported: "#5b6472",
    malformed: "#8b2e2e",
  };

  function assetUrl(path) {
    if (assetBase === ".") {
      return path;
    }
    return assetBase + "/" + path;
  }

  function hrefFor(item) {
    return isIndex ? item.href : assetBase + "/" + item.href;
  }

  function wrapLayout() {
    const header = document.querySelector("header");
    const main = document.querySelector("main");
    if (!header || !main) {
      return;
    }

    if (!document.getElementById("portfolio-nav")) {
      const nav = document.createElement("nav");
      nav.id = "portfolio-nav";
      nav.className = "portfolio-nav";
      nav.innerHTML = "<h2>Contents</h2><p>Loading...</p>";
      header.parentNode.insertBefore(nav, header);
    }

    if (!document.querySelector(".content-pane")) {
      const pane = document.createElement("div");
      pane.className = "content-pane";
      header.parentNode.insertBefore(pane, header);
      pane.appendChild(header);
      pane.appendChild(main);
    }

    body.classList.add("portfolio-layout");
  }

  function renderTree(tree) {
    const nav = document.getElementById("portfolio-nav");
    if (!nav) {
      return;
    }

    const heading = document.createElement("h2");
    heading.textContent = "Contents";

    const root = document.createElement("ul");
    root.className = "tree";

    const strategiesRoot = document.createElement("li");
    strategiesRoot.textContent = "strategies";

    const categoriesUl = document.createElement("ul");

    tree.categories.forEach(function (category) {
      const categoryLi = document.createElement("li");
      categoryLi.textContent = category.label.replace(/\/$/, "");

      const strategiesUl = document.createElement("ul");
      category.strategies.forEach(function (item) {
        const strategyLi = document.createElement("li");
        const link = document.createElement("a");
        link.href = hrefFor(item);
        link.textContent = item.label.replace(/\/$/, "");
        if (slug && item.slug === slug) {
          link.classList.add("active");
        }
        strategyLi.appendChild(link);
        strategiesUl.appendChild(strategyLi);
      });

      categoryLi.appendChild(strategiesUl);
      categoriesUl.appendChild(categoryLi);
    });

    strategiesRoot.appendChild(categoriesUl);
    root.appendChild(strategiesRoot);

    nav.replaceChildren(heading, root);
  }

  function formatNumber(value) {
    return Number(value || 0).toLocaleString("en-US");
  }

  function outcomeParts(data) {
    return [
      { key: "matches", label: "Match", value: Number(data.matches || 0) },
      { key: "mismatches", label: "Difference", value: Number(data.mismatches || 0) },
      { key: "unsupported", label: "Unsupported", value: Number(data.unsupported || 0) },
      { key: "malformed", label: "Malformed", value: Number(data.malformed || 0) },
    ];
  }

  function stackedBarSvg(parts, width, height) {
    const total = parts.reduce(function (sum, part) {
      return sum + part.value;
    }, 0);
    const ns = "http://www.w3.org/2000/svg";
    const svg = document.createElementNS(ns, "svg");
    svg.setAttribute("viewBox", "0 0 " + width + " " + height);
    svg.setAttribute("width", "100%");
    svg.setAttribute("height", String(height));
    svg.setAttribute("role", "img");
    svg.setAttribute("aria-label", "Decision replay outcome composition");
    svg.classList.add("outcome-bar");

    if (total <= 0) {
      const empty = document.createElementNS(ns, "rect");
      empty.setAttribute("x", "0");
      empty.setAttribute("y", "0");
      empty.setAttribute("width", String(width));
      empty.setAttribute("height", String(height));
      empty.setAttribute("fill", "#d7d7d7");
      svg.appendChild(empty);
      return svg;
    }

    let x = 0;
    parts.forEach(function (part) {
      if (part.value <= 0) {
        return;
      }
      const segmentWidth = (part.value / total) * width;
      const rect = document.createElementNS(ns, "rect");
      rect.setAttribute("x", String(x));
      rect.setAttribute("y", "0");
      rect.setAttribute("width", String(Math.max(segmentWidth, 0.5)));
      rect.setAttribute("height", String(height));
      rect.setAttribute("fill", OUTCOME_COLORS[part.key]);
      svg.appendChild(rect);
      x += segmentWidth;
    });
    return svg;
  }

  function legendList(parts) {
    const list = document.createElement("ul");
    list.className = "outcome-legend";
    parts.forEach(function (part) {
      const item = document.createElement("li");
      const swatch = document.createElement("span");
      swatch.className = "swatch";
      swatch.style.background = OUTCOME_COLORS[part.key];
      item.appendChild(swatch);
      item.appendChild(
        document.createTextNode(part.label + ": " + formatNumber(part.value))
      );
      list.appendChild(item);
    });
    return list;
  }

  function metricGrid(data) {
    const grid = document.createElement("div");
    grid.className = "metric-grid";
    [
      ["Rows", data.rows_seen],
      ["Match", data.matches],
      ["Difference", data.mismatches],
      ["Unsupported", data.unsupported],
      ["Malformed", data.malformed],
    ].forEach(function (pair) {
      const card = document.createElement("div");
      card.className = "metric-card";
      const label = document.createElement("div");
      label.className = "metric-label";
      label.textContent = pair[0];
      const value = document.createElement("div");
      value.className = "metric-value";
      value.textContent = formatNumber(pair[1]);
      card.appendChild(label);
      card.appendChild(value);
      grid.appendChild(card);
    });
    return grid;
  }

  function renderDisclaimer(text) {
    const note = document.createElement("p");
    note.className = "report-disclaimer";
    note.textContent = text;
    return note;
  }

  function renderDecisionReplaySection(replay) {
    const main = document.querySelector("main");
    if (!main || !replay) {
      return;
    }

    const existing = document.getElementById("decision-replay");
    if (existing) {
      existing.remove();
    }

    const section = document.createElement("section");
    section.id = "decision-replay";
    section.className = "decision-replay";

    const heading = document.createElement("h2");
    heading.textContent = "Decision Replay Outcome";
    section.appendChild(heading);

    section.appendChild(renderDisclaimer(replay.disclaimer));

    const status = document.createElement("p");
    status.className = "report-status";
    status.innerHTML =
      "<strong>" +
      (replay.status_label || replay.status) +
      "</strong>" +
      (replay.generated_at_utc
        ? " · report " + replay.generated_at_utc
        : "");
    section.appendChild(status);

    if (replay.source) {
      const source = document.createElement("p");
      source.className = "report-meta";
      source.textContent = "Source: " + replay.source;
      section.appendChild(source);
    }
    if (replay.config) {
      const config = document.createElement("p");
      config.className = "report-meta";
      config.textContent =
        "Frozen config: " +
        replay.config +
        (replay.config_fingerprint ? " (" + replay.config_fingerprint + ")" : "");
      section.appendChild(config);
    }

    const parts = outcomeParts(replay);
    section.appendChild(metricGrid(replay));
    section.appendChild(stackedBarSvg(parts, 640, 28));
    section.appendChild(legendList(parts));

    if (replay.non_router && replay.non_router.length) {
      const sub = document.createElement("h3");
      sub.textContent = "Family-specific coverage";
      section.appendChild(sub);
      replay.non_router.forEach(function (family) {
        const block = document.createElement("div");
        block.className = "family-coverage";
        const title = document.createElement("h4");
        title.textContent = family.family + " · " + family.source;
        block.appendChild(title);
        if (family.note) {
          const note = document.createElement("p");
          note.className = "report-meta";
          note.textContent = family.note;
          block.appendChild(note);
        }
        block.appendChild(metricGrid(family));
        const familyParts = outcomeParts(family);
        block.appendChild(stackedBarSvg(familyParts, 640, 22));
        block.appendChild(legendList(familyParts));
        section.appendChild(block);
      });
    }

    const interpretation = document.createElement("p");
    interpretation.className = "report-meta";
    interpretation.textContent =
      "Match means persisted policy inputs regenerated the recorded order or skip decision. Difference means the frozen configuration regenerated a different decision. Unsupported means the historical row did not persist enough input for an honest replay.";
    section.appendChild(interpretation);

    const statusSection = Array.from(main.querySelectorAll("section")).find(function (node) {
      const h2 = node.querySelector("h2");
      return h2 && h2.textContent === "Status and Outcome";
    });
    if (statusSection) {
      main.insertBefore(section, statusSection);
    } else {
      main.appendChild(section);
    }
  }

  function renderPortfolioReport(report) {
    const host = document.getElementById("portfolio-report");
    if (!host) {
      return;
    }
    host.replaceChildren();

    const heading = document.createElement("h2");
    heading.textContent = "Decision Replay Report";
    host.appendChild(heading);

    if (!report || !report.available) {
      host.appendChild(
        renderDisclaimer(
          (report && report.disclaimer) ||
            "Decision replay report unavailable. Run ./scripts/testing/run-decision-replay.sh then rebuild the portfolio site."
        )
      );
      return;
    }

    host.appendChild(renderDisclaimer(report.disclaimer));

    const intro = document.createElement("p");
    intro.textContent =
      "Portfolio rollup across all catalogued strategies. This page summarizes deterministic decision-replay coverage from recorded sessions. Open any strategy detail page for that strategy's own outcome chart and source metadata.";
    host.appendChild(intro);

    if (report.generated_at_utc) {
      const generated = document.createElement("p");
      generated.className = "report-meta";
      generated.textContent = "Generated: " + report.generated_at_utc;
      host.appendChild(generated);
    }

    const totals = report.totals || {};
    host.appendChild(metricGrid(totals));
    const parts = outcomeParts(totals);
    host.appendChild(stackedBarSvg(parts, 720, 32));
    host.appendChild(legendList(parts));

    const tableHeading = document.createElement("h3");
    tableHeading.textContent = "Per-strategy results";
    host.appendChild(tableHeading);

    const table = document.createElement("table");
    table.className = "report-table";
    table.innerHTML =
      "<thead><tr>" +
      "<th>Strategy</th><th>Status</th><th>Rows</th><th>Match</th>" +
      "<th>Difference</th><th>Unsupported</th><th>Malformed</th><th>Composition</th>" +
      "</tr></thead>";
    const tbody = document.createElement("tbody");
    (report.strategies || []).forEach(function (row) {
      const tr = document.createElement("tr");
      const nameTd = document.createElement("td");
      const link = document.createElement("a");
      link.href = row.href;
      link.textContent = row.slug;
      nameTd.appendChild(link);
      tr.appendChild(nameTd);

      [
        row.status_label || row.status,
        formatNumber(row.rows_seen),
        formatNumber(row.matches),
        formatNumber(row.mismatches),
        formatNumber(row.unsupported),
        formatNumber(row.malformed),
      ].forEach(function (cell) {
        const td = document.createElement("td");
        td.textContent = cell;
        tr.appendChild(td);
      });

      const chartTd = document.createElement("td");
      chartTd.appendChild(stackedBarSvg(outcomeParts(row), 120, 14));
      tr.appendChild(chartTd);
      tbody.appendChild(tr);
    });
    table.appendChild(tbody);
    host.appendChild(table);

    if (report.non_router_coverage && report.non_router_coverage.length) {
      const familyHeading = document.createElement("h3");
      familyHeading.textContent = "Non-router family coverage";
      host.appendChild(familyHeading);
      const familyList = document.createElement("ul");
      familyList.className = "family-list";
      report.non_router_coverage.forEach(function (family) {
        const item = document.createElement("li");
        item.textContent =
          family.family +
          " (" +
          family.source +
          "): " +
          formatNumber(family.rows_seen) +
          " rows · " +
          formatNumber(family.matches) +
          " match · " +
          formatNumber(family.unsupported) +
          " unsupported";
        familyList.appendChild(item);
      });
      host.appendChild(familyList);
    }

    const footer = document.createElement("p");
    footer.className = "report-meta";
    footer.textContent =
      "Select a strategy from the file tree for thesis, architecture, mathematics, and that strategy's own decision-replay detail.";
    host.appendChild(footer);
  }

  function renderCodeSamples(samples) {
    if (!samples || samples.length === 0) {
      return;
    }
    const main = document.querySelector("main");
    if (!main) {
      return;
    }

    const section = document.createElement("section");
    section.id = "code-samples";
    const heading = document.createElement("h2");
    heading.textContent = "Sample Code";
    section.appendChild(heading);

    samples.forEach(function (sample) {
      const block = document.createElement("div");
      block.className = "code-sample";

      const title = document.createElement("h3");
      title.textContent = sample.title;
      block.appendChild(title);

      const label = document.createElement("p");
      label.className = "code-label";
      label.textContent = sample.path;
      block.appendChild(label);

      const pre = document.createElement("pre");
      const code = document.createElement("code");
      code.textContent = sample.content;
      pre.appendChild(code);
      block.appendChild(pre);

      section.appendChild(block);
    });

    main.appendChild(section);
  }

  function renderLogPanel(meta) {
    const main = document.querySelector("main");
    if (!main) {
      return null;
    }

    const panel = document.createElement("section");
    panel.className = "log-panel";
    panel.id = "log-panel";

    const header = document.createElement("header");
    const title = document.createElement("h2");
    title.textContent = "Strategy Logs";
    header.appendChild(title);

    const status = document.createElement("span");
    status.className = "log-status";
    status.textContent = meta.source || "audit log";
    header.appendChild(status);
    panel.appendChild(header);

    const pre = document.createElement("pre");
    pre.id = "log-output";
    pre.textContent = meta.logs || "(no log lines available)";
    panel.appendChild(pre);

    main.appendChild(panel);
    return pre;
  }

  function updateLogOutput(pre, lines, updatedAt, live) {
    if (!pre) {
      return;
    }
    pre.textContent = lines.length ? lines.join("\n") : "(no log lines available)";
    const status = document.querySelector(".log-status");
    if (status) {
      const mode = live ? "live" : "snapshot";
      status.textContent =
        (status.dataset.source || "audit log") +
        " · " +
        mode +
        (updatedAt ? " · " + updatedAt : "");
    }
    const panel = document.getElementById("log-panel");
    if (panel) {
      panel.classList.toggle("live", !!live);
    }
  }

  async function fetchJson(url) {
    const response = await fetch(url, { cache: "no-store" });
    if (!response.ok) {
      throw new Error("HTTP " + response.status);
    }
    return response.json();
  }

  async function loadPortfolioReport() {
    if (!isIndex) {
      return;
    }
    try {
      const report = await fetchJson(assetUrl("data/portfolio-report.json"));
      renderPortfolioReport(report);
    } catch (_err) {
      renderPortfolioReport({
        available: false,
        disclaimer:
          "Decision replay report unavailable. Run ./scripts/testing/run-decision-replay.sh then rebuild the portfolio site.",
      });
    }
  }

  async function loadStrategyData() {
    if (!slug) {
      return;
    }

    let data;
    try {
      data = await fetchJson(assetUrl("data/" + slug + ".json"));
    } catch (_err) {
      return;
    }

    if (data.decision_replay) {
      renderDecisionReplaySection(data.decision_replay);
    }

    if (data.code_samples) {
      renderCodeSamples(data.code_samples);
    }

    const pre = renderLogPanel({
      source: data.log_source,
      logs: (data.logs || []).join("\n"),
    });
    const status = document.querySelector(".log-status");
    if (status && data.log_source) {
      status.dataset.source = data.log_source;
    }

    async function pollLogs() {
      const stamp = Date.now();
      try {
        const live = await fetchJson("/api/logs/" + slug + "?t=" + stamp);
        updateLogOutput(pre, live.lines || [], live.updated_at, true);
        return;
      } catch (_liveErr) {
        /* fall through to static snapshot */
      }

      try {
        const snapshot = await fetchJson(assetUrl("data/" + slug + ".json?t=" + stamp));
        updateLogOutput(pre, snapshot.logs || [], snapshot.updated_at, false);
      } catch (_snapErr) {
        /* keep existing content */
      }
    }

    pollLogs();
    window.setInterval(pollLogs, 3000);
  }

  async function init() {
    wrapLayout();

    try {
      const tree = await fetchJson(assetUrl("assets/tree.json"));
      renderTree(tree);
    } catch (_err) {
      const nav = document.getElementById("portfolio-nav");
      if (nav) {
        nav.innerHTML = "<h2>Contents</h2><p>Tree unavailable.</p>";
      }
    }

    await loadPortfolioReport();
    await loadStrategyData();
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", init);
  } else {
    init();
  }
})();
