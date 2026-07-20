const http = require("http");
const fs = require("fs");
const path = require("path");

const root = path.resolve(__dirname);
const port = Number(process.env.PORT || 8080);

const types = {
  ".html": "text/html; charset=utf-8",
  ".css": "text/css; charset=utf-8",
  ".js": "application/javascript; charset=utf-8",
  ".json": "application/json; charset=utf-8",
  ".png": "image/png",
  ".jpg": "image/jpeg",
  ".jpeg": "image/jpeg",
  ".gif": "image/gif",
  ".svg": "image/svg+xml",
  ".pdf": "application/pdf",
  ".mov": "video/quicktime",
  ".heic": "image/heic",
  ".ico": "image/x-icon",
  ".txt": "text/plain; charset=utf-8",
  ".md": "text/markdown; charset=utf-8",
};

function safeJoin(base, requestPath) {
  const decoded = decodeURIComponent(requestPath.split("?")[0]);
  const cleaned = path.normalize(decoded).replace(/^([/\\])+/, "");
  const full = path.join(base, cleaned);
  if (!full.startsWith(base)) return null;
  return full;
}

const server = http.createServer((req, res) => {
  let filePath = safeJoin(root, req.url || "/");
  if (!filePath) {
    res.writeHead(400);
    res.end("Bad request");
    return;
  }

  fs.stat(filePath, (err, stat) => {
    if (!err && stat.isDirectory()) {
      filePath = path.join(filePath, "index.html");
    }

    fs.readFile(filePath, (readErr, data) => {
      if (readErr) {
        res.writeHead(404, { "Content-Type": "text/plain; charset=utf-8" });
        res.end("Not found: " + req.url);
        return;
      }
      const ext = path.extname(filePath).toLowerCase();
      res.writeHead(200, { "Content-Type": types[ext] || "application/octet-stream" });
      res.end(data);
    });
  });
});

server.listen(port, "127.0.0.1", () => {
  console.log("Serving " + root);
  console.log("Open http://127.0.0.1:" + port + "/");
  console.log("Finance: http://127.0.0.1:" + port + "/finance/");
  console.log("Quant:   http://127.0.0.1:" + port + "/finance/quant/");
});
