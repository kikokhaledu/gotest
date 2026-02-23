const fs = require("node:fs");
const path = require("node:path");
const { spawnSync } = require("node:child_process");
const Module = require("node:module");

const projectRoot = path.resolve(__dirname, "..");
const testsDir = path.join(projectRoot, "test");
const testFiles = fs
  .readdirSync(testsDir)
  .filter((name) => name.endsWith(".test.js"))
  .sort((left, right) => left.localeCompare(right))
  .map((name) => path.join(testsDir, name));

const nodeMajorVersion = Number.parseInt(process.versions.node.split(".")[0], 10);
const supportsNativeNodeTest = Number.isFinite(nodeMajorVersion) && nodeMajorVersion >= 18;
const forceCompatRunner = process.env.FORCE_COMPAT_TEST_RUNNER === "1";

if (supportsNativeNodeTest && !forceCompatRunner) {
  const nativeRun = spawnSync(
    process.execPath,
    ["--test", "--test-reporter=spec", ...testFiles],
    {
      stdio: "inherit",
      cwd: projectRoot,
      env: process.env,
    }
  );

  if (nativeRun.error) {
    console.error("[tests] native node:test execution failed", nativeRun.error);
    process.exit(1);
  }
  process.exit(nativeRun.status ?? 1);
}

const registeredTests = [];
const skippedTests = [];

function registerTest(name, fn, filePath, skip) {
  if (typeof name !== "string" || name.trim() === "") {
    throw new TypeError(`Invalid test name in ${filePath}`);
  }
  if (typeof fn !== "function") {
    throw new TypeError(`Test "${name}" in ${filePath} is missing a function body`);
  }
  if (skip) {
    skippedTests.push({ name, filePath });
    return;
  }
  registeredTests.push({ name, fn, filePath });
}

function createTestAPI(filePath) {
  const testFn = (name, fn) => registerTest(name, fn, filePath, false);
  testFn.skip = (name, fn) => registerTest(name, fn || (() => {}), filePath, true);
  testFn.todo = (name) => skippedTests.push({ name, filePath });
  testFn.only = (name, fn) => registerTest(name, fn, filePath, false);
  return testFn;
}

async function runFallbackHarness() {
  console.log(`[tests] Node ${process.versions.node} detected; using compatibility test harness`);

  const originalLoad = Module._load;
  const apiByFile = new Map();
  let activeFile = null;

  Module._load = function patchedLoad(request, parent, isMain) {
    if (request === "node:test") {
      if (!activeFile) {
        throw new Error("node:test requested outside a known test file");
      }
      if (!apiByFile.has(activeFile)) {
        apiByFile.set(activeFile, createTestAPI(activeFile));
      }
      return apiByFile.get(activeFile);
    }
    return originalLoad(request, parent, isMain);
  };

  try {
    for (const filePath of testFiles) {
      activeFile = filePath;
      delete require.cache[filePath];
      require(filePath);
    }
  } finally {
    activeFile = null;
    Module._load = originalLoad;
  }

  let passed = 0;
  let failed = 0;
  const startedAt = Date.now();

  for (const testCase of registeredTests) {
    const testStart = Date.now();
    try {
      await Promise.resolve(testCase.fn());
      passed += 1;
      console.log(`✔ ${testCase.name} (${Date.now() - testStart}ms)`);
    } catch (error) {
      failed += 1;
      console.error(`✖ ${testCase.name}`);
      console.error(`  at ${path.relative(projectRoot, testCase.filePath)}`);
      console.error(error && error.stack ? error.stack : error);
    }
  }

  for (const skipped of skippedTests) {
    console.log(`- ${skipped.name} (skipped)`);
  }

  console.log(`ℹ tests ${registeredTests.length + skippedTests.length}`);
  console.log(`ℹ pass ${passed}`);
  console.log(`ℹ fail ${failed}`);
  console.log(`ℹ skipped ${skippedTests.length}`);
  console.log(`ℹ duration_ms ${Date.now() - startedAt}`);

  if (failed > 0) {
    process.exit(1);
  }
}

runFallbackHarness().catch((error) => {
  console.error("[tests] fallback harness failed");
  console.error(error && error.stack ? error.stack : error);
  process.exit(1);
});
