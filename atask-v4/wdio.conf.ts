import { spawn, type ChildProcess } from "child_process";
import { platform } from "os";
import { resolve } from "path";

// Path to the built Tauri debug binary
function getAppBinaryPath(): string {
  const target = resolve("src-tauri/target/debug");
  if (platform() === "darwin") {
    return resolve(
      target,
      "bundle/macos/atask.app/Contents/MacOS/atask-v4",
    );
  } else if (platform() === "win32") {
    return resolve(target, "atask-v4.exe");
  }
  return resolve(target, "atask-v4");
}

// Use tauri-wd on macOS (WKWebView), tauri-driver on Linux (WebKitGTK)
function getDriverCommand(): string {
  if (platform() === "darwin") {
    return "tauri-wd";
  }
  return "tauri-driver";
}

let driver: ChildProcess;

export const config = {
  runner: "local",
  port: 4444,
  hostname: "localhost",

  specs: ["./tests/e2e/**/*.test.ts"],
  exclude: [],

  maxInstances: 1,
  capabilities: [
    {
      "tauri:options": {
        binary: getAppBinaryPath(),
      },
    } as Record<string, unknown>,
  ],

  logLevel: "warn",
  bail: 0,
  specFileRetries: 0,
  waitforTimeout: 10000,
  connectionRetryTimeout: 120000,
  connectionRetryCount: 3,

  framework: "mocha",
  reporters: ["spec"],

  mochaOpts: {
    ui: "bdd",
    timeout: 60000,
  },

  // Start the WebDriver bridge before the test session
  onPrepare: function () {
    const cmd = getDriverCommand();
    driver = spawn(cmd, ["--port", "4444"], {
      stdio: [null, process.stdout, process.stderr],
    });

    // Wait for the driver to bind to port 4444
    return new Promise<void>((resolve) => {
      setTimeout(resolve, 3000);
    });
  },

  // Stop the driver after tests complete
  onComplete: function () {
    driver.kill();
  },

  // Give the Tauri app time to launch and render
  before: async function () {
    await browser.pause(4000);
  },
};
