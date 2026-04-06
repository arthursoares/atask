import {
  waitForAppReady,
  resetDatabase,
  navigateTo,
  createTaskViaUI,
  getTaskTitles,
  clickTask,
  pressKeys,
  isCommandPaletteOpen,
  isDetailPanelVisible,
  elementExists,
  getSidebarLabels,
  isTaskCompleted,
} from "./helpers";

describe("UI Fixes", () => {
  before(async () => {
    await waitForAppReady();
    await navigateTo("Inbox");
  });

  beforeEach(async () => {
    await resetDatabase();
    await waitForAppReady();
  });

  describe("Cmd+K opens command palette", () => {
    it("should open command palette with Cmd+K", async () => {
      await browser.execute(() => {
        (document.activeElement as HTMLElement)?.blur();
      });
      await pressKeys("k", true);
      await browser.pause(300);
      expect(await isCommandPaletteOpen()).toBe(true);
    });

    it("should close command palette with Escape", async () => {
      // Cmd+K can't close it because the palette input has focus (isEditingText)
      // Use Escape which the palette handles directly
      await browser.execute(() => {
        const input = document.querySelector(".cmd-input") as HTMLInputElement;
        if (input) {
          input.dispatchEvent(
            new KeyboardEvent("keydown", { key: "Escape", code: "Escape", bubbles: true }),
          );
        }
      });
      await browser.pause(300);
      expect(await isCommandPaletteOpen()).toBe(false);
    });
  });

  describe("Cmd+Shift+P opens command palette", () => {
    // TODO: synthetic KeyboardEvent with shift+meta doesn't propagate reliably in WebDriver
    it.skip("should open command palette with Cmd+Shift+P", async () => {
      await browser.execute(() => {
        (document.activeElement as HTMLElement)?.blur();
      });
      await pressKeys("p", true, true);
      await browser.pause(300);
      expect(await isCommandPaletteOpen()).toBe(true);
    });
  });

  describe("Shift+Cmd+C completes task", () => {
    it("should create and complete a task with Shift+Cmd+C", async () => {
      await createTaskViaUI("Complete Via Shortcut");
      await clickTask("Complete Via Shortcut");
      await browser.execute(() => {
        (document.activeElement as HTMLElement)?.blur();
      });
      await browser.pause(100);

      await pressKeys("c", true, true);
      await browser.pause(300);
      expect(await isTaskCompleted("Complete Via Shortcut")).toBe(true);
    });
  });

  describe("Cmd+N focuses new task input", () => {
    it("should focus the new task input on Cmd+N", async () => {
      await browser.execute(() => {
        (document.activeElement as HTMLElement)?.blur();
      });
      await browser.pause(100);

      await pressKeys("n", true);
      await browser.pause(500);

      // The new task row should be in editing mode (input visible)
      const hasInput = await browser.execute(() => {
        const row = document.querySelector(".new-task-inline");
        if (!row) return false;
        return row.querySelector("input") !== null;
      });
      expect(hasInput).toBe(true);

      // Press Escape to exit editing
      await browser.execute(() => {
        const input = document.querySelector(".new-task-inline input") as HTMLInputElement;
        if (input) {
          input.dispatchEvent(
            new KeyboardEvent("keydown", { key: "Escape", code: "Escape", bubbles: true }),
          );
        }
      });
      await browser.pause(200);
    });
  });

  describe("Window drag region", () => {
    it("should have drag region on toolbar", async () => {
      const hasDragRegion = await browser.execute(() => {
        const toolbar = document.querySelector(".app-toolbar");
        return toolbar?.hasAttribute("data-tauri-drag-region") ?? false;
      });
      expect(hasDragRegion).toBe(true);
    });

    it("should have drag region on sidebar toolbar", async () => {
      const hasDragRegion = await browser.execute(() => {
        const toolbar = document.querySelector(".sidebar-toolbar");
        return toolbar?.hasAttribute("data-tauri-drag-region") ?? false;
      });
      expect(hasDragRegion).toBe(true);
    });
  });

  describe("No custom traffic lights", () => {
    it("should not have custom traffic light elements", async () => {
      const hasTrafficLights = await browser.execute(() => {
        return document.querySelector(".sidebar-traffic-lights") !== null;
      });
      expect(hasTrafficLights).toBe(false);
    });
  });

  describe("Areas visible without projects", () => {
    it("should create an area", async () => {
      // Click the "+ New Area" button
      await browser.execute(() => {
        const items = document.querySelectorAll(".sidebar-item");
        for (const item of items) {
          if (item.textContent?.includes("New Area")) {
            (item as HTMLElement).click();
            return;
          }
        }
      });
      await browser.pause(500);
    });

    it("should show area in sidebar even without projects", async () => {
      const hasAreaLabel = await browser.execute(() => {
        const labels = document.querySelectorAll(".sidebar-group-label");
        return labels.length > 0;
      });
      expect(hasAreaLabel).toBe(true);
    });
  });

  describe("Area rename via context menu", () => {
    it("should open area context menu with right-click", async () => {
      await browser.execute(() => {
        const labels = document.querySelectorAll(".sidebar-group-label");
        for (const label of labels) {
          label.dispatchEvent(
            new MouseEvent("contextmenu", { bubbles: true, clientX: 200, clientY: 200 }),
          );
          return;
        }
      });
      await browser.pause(200);

      // Context menu should show "Rename"
      const hasRename = await browser.execute(() => {
        const els = document.querySelectorAll("[style*='position: fixed']");
        for (const el of els) {
          if (el.textContent?.includes("Rename")) return true;
        }
        return false;
      });
      expect(hasRename).toBe(true);

      // Close it
      await browser.execute(() => {
        document.dispatchEvent(new MouseEvent("mousedown", { bubbles: true }));
      });
      await browser.pause(200);
    });
  });

  describe("Project pill hidden in project view", () => {
    it("should create a project and task", async () => {
      await pressKeys("O", true, true);
      await browser.pause(300);
      await browser.execute(() => {
        const items = document.querySelectorAll(".cmd-item");
        for (const item of items) {
          if (item.textContent?.includes("New Project")) {
            (item as HTMLElement).click();
            return;
          }
        }
      });
      await browser.pause(500);

      await navigateTo("New Project");
      await browser.pause(300);
      await createTaskViaUI("Project Task No Pill");
    });

    it("should NOT show project pill on tasks in project view", async () => {
      const hasProjectPill = await browser.execute(() => {
        const pills = document.querySelectorAll(".task-project-pill");
        return pills.length;
      });
      expect(hasProjectPill).toBe(0);
    });
  });

  after(async () => {
    await navigateTo("Inbox");
  });
});
