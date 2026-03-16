import { mkdirSync, existsSync } from "node:fs";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { spawn } from "node:child_process";
import net from "node:net";

const __dirname = dirname(fileURLToPath(import.meta.url));
const projectRoot = resolve(__dirname, "..");
const distPath = join(projectRoot, "frontend", "dist");

const [, , outputPathArg, previewArg = "running", widthArg = "1440", heightArg = "1600"] = process.argv;

if (!outputPathArg) {
  throw new Error("Usage: node scripts/capture-preview-screenshot.mjs <outputPath> [preview] [width] [height]");
}

if (!existsSync(distPath)) {
  throw new Error(`前端构建产物不存在：${distPath}`);
}

const browserCandidates = [
  "C:\\Program Files (x86)\\Microsoft\\EdgeCore\\131.0.2903.86\\msedge.exe",
  "C:\\Program Files (x86)\\Microsoft\\EdgeWebView\\Application\\131.0.2903.86\\msedge.exe",
  "C:\\Users\\Administrator\\AppData\\Local\\Google\\Chrome\\Application\\chrome.exe"
];

const browserPath = browserCandidates.find((candidate) => existsSync(candidate));
if (!browserPath) {
  throw new Error("未找到可用的 Chromium 浏览器可执行文件。");
}

const outputPath = resolve(projectRoot, outputPathArg);
mkdirSync(dirname(outputPath), { recursive: true });

const port = 34115;
const width = Number.parseInt(widthArg, 10);
const height = Number.parseInt(heightArg, 10);
const previewUrl = `http://127.0.0.1:${port}/?preview=${previewArg}`;

function waitForPort(portNumber, timeoutMs = 10000) {
  const startedAt = Date.now();

  return new Promise((resolvePromise, rejectPromise) => {
    const attempt = () => {
      const socket = net.createConnection({ host: "127.0.0.1", port: portNumber });

      socket.once("connect", () => {
        socket.destroy();
        resolvePromise(undefined);
      });

      socket.once("error", () => {
        socket.destroy();
        if (Date.now() - startedAt >= timeoutMs) {
          rejectPromise(new Error(`本地预览服务未能在端口 ${portNumber} 上启动。`));
          return;
        }

        setTimeout(attempt, 250);
      });
    };

    attempt();
  });
}

const pythonServer = spawn("python", ["-m", "http.server", String(port), "--bind", "127.0.0.1", "--directory", distPath], {
  cwd: projectRoot,
  stdio: "ignore",
  windowsHide: true
});

try {
  await waitForPort(port);

  await new Promise((resolvePromise, rejectPromise) => {
    const browser = spawn(
      browserPath,
      [
        "--headless",
        "--disable-gpu",
        "--hide-scrollbars",
        `--window-size=${width},${height}`,
        "--virtual-time-budget=3000",
        `--screenshot=${outputPath}`,
        previewUrl
      ],
      {
        cwd: projectRoot,
        stdio: "ignore",
        windowsHide: true
      }
    );

    browser.once("error", rejectPromise);
    browser.once("exit", () => resolvePromise(undefined));
  });

  if (!existsSync(outputPath)) {
    throw new Error(`截图失败，未生成文件：${outputPath}`);
  }

  console.log(outputPath);
} finally {
  pythonServer.kill("SIGTERM");
}
