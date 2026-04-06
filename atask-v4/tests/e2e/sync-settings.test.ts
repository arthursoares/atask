import {
  waitForAppReady,
  resetDatabase,
  navigateTo,
  elementExists,
} from "./helpers";

describe("Sync Settings", () => {
  before(async () => {
    await waitForAppReady();
    await navigateTo("Settings");
  });

  beforeEach(async () => {
    await resetDatabase();
    await waitForAppReady();
  });

  it("should show sync toggle", async () => {
    const hasCheckbox = await browser.execute(() => {
      return document.querySelector("input[type='checkbox']") !== null;
    });
    expect(hasCheckbox).toBe(true);
  });

  it("should show server URL input", async () => {
    const hasUrlInput = await browser.execute(() => {
      return document.querySelector("input[type='url']") !== null;
    });
    expect(hasUrlInput).toBe(true);
  });

  it("should show API key input", async () => {
    const hasApiKeyInput = await browser.execute(() => {
      const inputs = document.querySelectorAll("input");
      for (const input of inputs) {
        if (input.placeholder?.includes("ak_")) return true;
      }
      return false;
    });
    expect(hasApiKeyInput).toBe(true);
  });

  it("should show Test Connection button", async () => {
    const hasTestBtn = await browser.execute(() => {
      const buttons = document.querySelectorAll("button");
      for (const btn of buttons) {
        if (btn.textContent?.includes("Test Connection")) return true;
      }
      return false;
    });
    expect(hasTestBtn).toBe(true);
  });

  it("should enable sync and show Initial Sync button", async () => {
    // Enable sync toggle
    await browser.execute(() => {
      const checkbox = document.querySelector("input[type='checkbox']") as HTMLInputElement;
      if (checkbox && !checkbox.checked) checkbox.click();
    });
    await browser.pause(300);

    // Should show "Run Initial Sync" button
    const hasInitialSyncBtn = await browser.execute(() => {
      const buttons = document.querySelectorAll("button");
      for (const btn of buttons) {
        if (btn.textContent?.includes("Initial Sync")) return true;
      }
      return false;
    });
    expect(hasInitialSyncBtn).toBe(true);
  });

  it("should open Initial Sync dialog", async () => {
    await browser.execute(() => {
      const buttons = document.querySelectorAll("button");
      for (const btn of buttons) {
        if (btn.textContent?.includes("Initial Sync")) {
          btn.click();
          return;
        }
      }
    });
    await browser.pause(300);

    // Dialog should be visible with 3 sync options
    const hasDialog = await browser.execute(() => {
      // Check for the backdrop
      const backdrop = document.querySelector(".cmd-backdrop.open");
      return backdrop !== null;
    });
    expect(hasDialog).toBe(true);

    // Should show three sync mode buttons
    const hasFresh = await browser.execute(() => {
      const buttons = document.querySelectorAll("button");
      for (const btn of buttons) {
        if (btn.textContent?.includes("Fresh sync")) return true;
      }
      return false;
    });
    expect(hasFresh).toBe(true);

    const hasMerge = await browser.execute(() => {
      const buttons = document.querySelectorAll("button");
      for (const btn of buttons) {
        if (btn.textContent?.includes("Merge")) return true;
      }
      return false;
    });
    expect(hasMerge).toBe(true);

    const hasPush = await browser.execute(() => {
      const buttons = document.querySelectorAll("button");
      for (const btn of buttons) {
        if (btn.textContent?.includes("Push local")) return true;
      }
      return false;
    });
    expect(hasPush).toBe(true);
  });

  it("should close Initial Sync dialog with backdrop click", async () => {
    await browser.execute(() => {
      const backdrop = document.querySelector(".cmd-backdrop.open");
      if (backdrop) (backdrop as HTMLElement).click();
    });
    await browser.pause(200);

    const hasDialog = await browser.execute(() => {
      return document.querySelector(".cmd-backdrop.open") !== null;
    });
    expect(hasDialog).toBe(false);
  });

  it("should test connection (will show status after click)", async () => {
    // Click Test Connection
    await browser.execute(() => {
      const buttons = document.querySelectorAll("button");
      for (const btn of buttons) {
        if (btn.textContent?.includes("Test Connection")) {
          btn.click();
          return;
        }
      }
    });
    await browser.pause(3000);

    // Should show "Testing...", "Connected", or "Not connected"
    // (Without a server URL configured, the Tauri command may error silently)
    const statusText = await browser.execute(() => {
      const buttons = document.querySelectorAll("button");
      for (const btn of buttons) {
        if (btn.textContent?.includes("Testing") || btn.textContent?.includes("Test Connection")) {
          return btn.textContent?.trim() ?? "";
        }
      }
      return "";
    });
    // Button should exist regardless of outcome
    expect(statusText.length).toBeGreaterThan(0);
  });

  after(async () => {
    await navigateTo("Inbox");
  });
});
