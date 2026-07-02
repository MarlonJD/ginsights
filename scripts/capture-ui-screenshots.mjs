#!/usr/bin/env node
import { createRequire } from "node:module";
import fs from "node:fs/promises";
import os from "node:os";
import path from "node:path";

const sections = [
  { name: "pulse", fragment: "" },
  { name: "commits", fragment: "#commits" },
  { name: "contributors", fragment: "#contributors" },
  { name: "code-frequency", fragment: "#code-frequency" },
  { name: "files", fragment: "#files" },
  { name: "languages", fragment: "#languages" },
  { name: "health", fragment: "#health" },
  { name: "provenance", fragment: "#provenance" },
];

const viewports = [
  { name: "desktop", width: 1920, height: 1080 },
  { name: "mobile", width: 390, height: 844, isMobile: true },
];

function usage() {
  return `Capture ginsights UI screenshots with Playwright.

Usage:
  node scripts/capture-ui-screenshots.mjs --base-url URL --out DIR [--readme-asset FILE]

Environment:
  PLAYWRIGHT_NODE_MODULES  Optional node_modules directory containing Playwright.
`;
}

function parseArgs(argv) {
  const out = { baseUrl: "", outDir: "", readmeAsset: "" };
  for (let i = 0; i < argv.length; i++) {
    const arg = argv[i];
    switch (arg) {
      case "--help":
      case "-h":
        console.log(usage());
        process.exit(0);
        break;
      case "--base-url":
        out.baseUrl = argv[++i] || "";
        break;
      case "--out":
        out.outDir = argv[++i] || "";
        break;
      case "--readme-asset":
        out.readmeAsset = argv[++i] || "";
        break;
      default:
        throw new Error(`unknown argument: ${arg}`);
    }
  }
  if (!out.baseUrl || !out.outDir) {
    throw new Error("--base-url and --out are required");
  }
  return out;
}

function requireFromNodeModules(moduleName, nodeModulesDir) {
  const require = createRequire(path.join(nodeModulesDir, "package.json"));
  return require(moduleName);
}

function loadPlaywright() {
  const requireLocal = createRequire(import.meta.url);
  try {
    return requireLocal("playwright");
  } catch {
    // Fall through to explicit module roots below.
  }

  const roots = [
    process.env.PLAYWRIGHT_NODE_MODULES,
    path.join(os.homedir(), ".cache/codex-runtimes/codex-primary-runtime/dependencies/node/node_modules"),
  ].filter(Boolean);

  for (const root of roots) {
    try {
      return requireFromNodeModules("playwright", root);
    } catch {
      // Try the next root.
    }
  }

  throw new Error(
    "Playwright package not found. Install it locally or set PLAYWRIGHT_NODE_MODULES to a node_modules directory containing playwright.",
  );
}

function cleanBaseUrl(url) {
  return url.replace(/\/+$/, "");
}

async function capturePage(page, baseUrl, outDir, viewport, section, readmeAsset) {
  const url = `${baseUrl}/${section.fragment}`;
  await page.setViewportSize({ width: viewport.width, height: viewport.height });
  await page.goto(url, { waitUntil: "networkidle" });

  if (section.fragment) {
    await page.evaluate(
      ({ fragment, useStickyOffset }) => {
        const el = document.querySelector(fragment);
        if (!el) return;
        const sticky = useStickyOffset ? document.querySelector(".topbar") : null;
        const stickyHeight = sticky ? sticky.getBoundingClientRect().height : 0;
        const top = el.getBoundingClientRect().top + window.scrollY - stickyHeight - 16;
        window.scrollTo(0, Math.max(0, top));
      },
      { fragment: section.fragment, useStickyOffset: viewport.name === "mobile" },
    );
  } else {
    await page.evaluate(() => window.scrollTo(0, 0));
  }
  await page.waitForTimeout(150);

  const file = path.join(outDir, `${viewport.name}-${viewport.width}x${viewport.height}-${section.name}.png`);
  await page.screenshot({ path: file, fullPage: false });

  if (readmeAsset && viewport.name === "desktop" && section.name === "pulse") {
    await page.screenshot({ path: readmeAsset, type: "jpeg", quality: 88, fullPage: false });
  }

  return file;
}

async function main() {
  const args = parseArgs(process.argv.slice(2));
  const { chromium } = loadPlaywright();
  const outDir = path.resolve(args.outDir);
  const baseUrl = cleanBaseUrl(args.baseUrl);

  await fs.mkdir(outDir, { recursive: true });
  for (const entry of await fs.readdir(outDir)) {
    if (entry.endsWith(".png") || entry === "manifest.json") {
      await fs.rm(path.join(outDir, entry), { force: true });
    }
  }
  if (args.readmeAsset) {
    await fs.mkdir(path.dirname(path.resolve(args.readmeAsset)), { recursive: true });
  }

  const userDataDir = await fs.mkdtemp(path.join(os.tmpdir(), "ginsights-playwright-"));
  const captures = [];
  const consoleMessages = [];
  const browser = await chromium.launchPersistentContext(userDataDir, {
    headless: true,
    viewport: { width: 1920, height: 1080 },
  });

  try {
    const page = await browser.newPage();
    page.on("console", (msg) => {
      if (msg.type() === "error" || msg.type() === "warning") {
        consoleMessages.push({ type: msg.type(), text: msg.text() });
      }
    });

    await page.goto(`${baseUrl}/`, { waitUntil: "networkidle" });
    const meaningful = await page.locator("text=Repository atlas").count();
    if (meaningful < 1) {
      throw new Error("Rendered page did not contain expected Repository atlas text.");
    }

    for (const viewport of viewports) {
      for (const section of sections) {
        const file = await capturePage(page, baseUrl, outDir, viewport, section, args.readmeAsset);
        captures.push({ viewport: viewport.name, section: section.name, file });
        console.log(`captured ${file}`);
      }
    }
  } finally {
    await browser.close();
    await fs.rm(userDataDir, { recursive: true, force: true });
  }

  const manifest = {
    generatedAt: new Date().toISOString(),
    baseUrl,
    viewports,
    sections: sections.map((section) => section.name),
    consoleMessages,
    captures,
  };
  await fs.writeFile(path.join(outDir, "manifest.json"), `${JSON.stringify(manifest, null, 2)}\n`);

  if (args.readmeAsset) {
    console.log(`updated ${path.resolve(args.readmeAsset)}`);
  }
  console.log(`screenshots written to ${outDir}`);
}

main().catch((err) => {
  console.error(`capture-ui-screenshots: ${err.message}`);
  process.exit(1);
});
