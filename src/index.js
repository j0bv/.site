function gridCellDimensions() {
  const element = document.createElement("div");
  element.style.position = "fixed";
  element.style.height = "var(--line-height)";
  element.style.width = "1ch";
  document.body.appendChild(element);
  const rect = element.getBoundingClientRect();
  document.body.removeChild(element);
  return { width: rect.width, height: rect.height };
}

// Add padding to each media to maintain grid.
function adjustMediaPadding() {
  const cell = gridCellDimensions();

  function setHeightFromRatio(media, ratio) {
      const rect = media.getBoundingClientRect();
      const realHeight = rect.width / ratio;
      const diff = cell.height - (realHeight % cell.height);
      media.style.setProperty("padding-bottom", `${diff}px`);
  }

  function setFallbackHeight(media) {
      const rect = media.getBoundingClientRect();
      const height = Math.round((rect.width / 2) / cell.height) * cell.height;
      media.style.setProperty("height", `${height}px`);
  }

  function onMediaLoaded(media) {
    var width, height;
    switch (media.tagName) {
      case "IMG":
        width = media.naturalWidth;
        height = media.naturalHeight;
        break;
      case "VIDEO":
        width = media.videoWidth;
        height = media.videoHeight;
        break;
    }
    if (width > 0 && height > 0) {
      setHeightFromRatio(media, width / height);
    } else {
      setFallbackHeight(media);
    }
  }

  const medias = document.querySelectorAll("img, video");
  for (media of medias) {
    switch (media.tagName) {
      case "IMG":
        if (media.complete) {
          onMediaLoaded(media);
        } else {
          media.addEventListener("load", () => onMediaLoaded(media));
          media.addEventListener("error", function() {
              setFallbackHeight(media);
          });
        }
        break;
      case "VIDEO":
        switch (media.readyState) {
          case HTMLMediaElement.HAVE_CURRENT_DATA:
          case HTMLMediaElement.HAVE_FUTURE_DATA:
          case HTMLMediaElement.HAVE_ENOUGH_DATA:
            onMediaLoaded(media);
            break;
          default:
            media.addEventListener("loadeddata", () => onMediaLoaded(media));
            media.addEventListener("error", function() {
              setFallbackHeight(media);
            });
            break;
        }
        break;
    }
  }
}

adjustMediaPadding();
window.addEventListener("load", adjustMediaPadding);
window.addEventListener("resize", adjustMediaPadding);

document.body.addEventListener("click", function (e) {
  if (e.target.classList.contains("scroll-to-top")) {
    e.preventDefault();
    window.scrollTo({ top: 0, behavior: "smooth" });
  }
});

// Fix links for file:// protocol (when opening HTML files directly)
// Convert clean URLs like /resume to relative paths based on current file location
(function fixLinksForFileProtocol() {
  // Check if we're running from file:// protocol
  if (window.location.protocol === 'file:') {
    // Determine current file depth (how many directories deep we are)
    const currentPath = window.location.pathname;
    const depth = currentPath.split('/').filter(p => p && !p.endsWith('.html')).length - 1;
    const relativePrefix = depth > 0 ? '../'.repeat(depth) : '';
    
    // Find all links that start with / (absolute paths)
    const links = document.querySelectorAll('a[href^="/"]');
    links.forEach(link => {
      const href = link.getAttribute('href');
      // Skip if it's already a full URL or special protocol
      if (href.startsWith('http://') || href.startsWith('https://') || href.startsWith('mailto:')) {
        return;
      }
      
      // Handle root/home link
      if (href === '/') {
        link.setAttribute('href', `${relativePrefix}index.html`);
        return;
      }
      
      // Handle design-guide
      if (href === '/design-guide') {
        link.setAttribute('href', `${relativePrefix}design-guide.html`);
        return;
      }
      
      // Convert /resume -> jobv/resume.html (or ../jobv/resume.html if in subdirectory)
      // Convert /code/newslogic -> jobv/code/newslogic.html
      const path = href.substring(1); // Remove leading /
      const parts = path.split('/');
      if (parts.length === 1) {
        // Single level: /resume -> jobv/resume.html
        link.setAttribute('href', `${relativePrefix}jobv/${parts[0]}.html`);
      } else {
        // Nested: /code/newslogic -> jobv/code/newslogic.html
        const lastPart = parts[parts.length - 1];
        const pathParts = parts.slice(0, -1);
        link.setAttribute('href', `${relativePrefix}jobv/${pathParts.join('/')}/${lastPart}.html`);
      }
    });
  }
})();

function checkOffsets() {
  const ignoredTagNames = new Set([
    "THEAD",
    "TBODY",
    "TFOOT",
    "TR",
    "TD",
    "TH",
  ]);
  const cell = gridCellDimensions();
  const elements = document.querySelectorAll("body :not(.debug-grid, .debug-toggle)");
  for (const element of elements) {
    if (ignoredTagNames.has(element.tagName)) {
      continue;
    }
    const rect = element.getBoundingClientRect();
    if (rect.width === 0 && rect.height === 0) {
      continue;
    }
    const top = rect.top + window.scrollY;
    const left = rect.left + window.scrollX;
    const offset = top % (cell.height / 2);
    if(offset > 0) {
      element.classList.add("off-grid");
      console.error("Incorrect vertical offset for", element, "with remainder", top % cell.height, "when expecting divisible by", cell.height / 2);
    } else {
      element.classList.remove("off-grid");
    }
  }
}

