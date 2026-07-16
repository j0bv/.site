(function () {
  "use strict";

  const body = document.body;
  const assetBase = body.dataset.assetBase || ".";
  const slug = body.dataset.slug || "";
  const isIndex = body.dataset.page === "index";

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

    await loadStrategyData();
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", init);
  } else {
    init();
  }
})();
