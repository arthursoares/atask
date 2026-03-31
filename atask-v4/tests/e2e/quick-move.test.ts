import {
  waitForAppReady,
  navigateTo,
  createTaskViaUI,
  clickTask,
  pressKeys,
  getDetailFieldValue,
  isDetailPanelVisible,
  clickCommandPaletteItem,
  getSidebarLabels,
} from "./helpers";

describe("QuickMovePicker", () => {
  before(async () => {
    await waitForAppReady();
    await navigateTo("Inbox");

    // Ensure a project exists
    await pressKeys("O", true, true);
    await browser.pause(300);
    await clickCommandPaletteItem("New Project");
    await browser.pause(500);
    await navigateTo("Inbox");
  });

  it("should create a task to move", async () => {
    await createTaskViaUI("QuickMove Test Task");
    await clickTask("QuickMove Test Task");
    expect(await isDetailPanelVisible()).toBe(true);
  });

  it("should open QuickMovePicker with Shift+Cmd+M", async () => {
    await browser.execute(() => {
      (document.activeElement as HTMLElement)?.blur();
    });
    await browser.pause(100);
    await pressKeys("m", true, true);
    await browser.pause(300);

    // QuickMovePicker should be visible (uses cmd-backdrop.open)
    const hasQuickMove = await browser.execute(() => {
      const backdrops = document.querySelectorAll(".cmd-backdrop.open");
      return backdrops.length > 0;
    });
    expect(hasQuickMove).toBe(true);
  });

  it("should show search input and project list", async () => {
    // Check for the search input
    const hasInput = await browser.execute(() => {
      const inputs = document.querySelectorAll(".cmd-input");
      for (const input of inputs) {
        if ((input as HTMLInputElement).placeholder?.includes("Move to project")) return true;
      }
      return false;
    });
    expect(hasInput).toBe(true);

    // Check for "No Project" option
    const hasNoProject = await browser.execute(() => {
      const els = document.querySelectorAll(".cmd-item, .cmd-item-label");
      for (const el of els) {
        if (el.textContent?.includes("No Project") || el.textContent?.includes("Inbox")) return true;
      }
      // Also check divs that might contain the option
      const divs = document.querySelectorAll("div");
      for (const div of divs) {
        if (div.textContent?.includes("No Project")) return true;
      }
      return false;
    });
    expect(hasNoProject).toBe(true);
  });

  it("should close with Escape", async () => {
    await browser.execute(() => {
      const input = document.querySelector(".cmd-input") as HTMLInputElement;
      if (input) {
        input.dispatchEvent(
          new KeyboardEvent("keydown", { key: "Escape", code: "Escape", bubbles: true }),
        );
      }
    });
    await browser.pause(300);

    const hasPicker = await browser.execute(() => {
      const backdrops = document.querySelectorAll(".cmd-backdrop.open");
      return backdrops.length > 0;
    });
    expect(hasPicker).toBe(false);
  });

  after(async () => {
    await pressKeys("Escape");
    await navigateTo("Inbox");
  });
});
