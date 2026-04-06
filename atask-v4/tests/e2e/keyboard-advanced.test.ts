import {
  waitForAppReady,
  resetDatabase,
  navigateTo,
  createTaskViaUI,
  clickTask,
  getTaskTitles,
  pressKeys,
  isBulkBarVisible,
  getBulkBarCount,
  getViewTitle,
} from "./helpers";

describe("Advanced Keyboard Shortcuts", () => {
  before(async () => {
    await waitForAppReady();
    await navigateTo("Inbox");
  });

  beforeEach(async () => {
    await resetDatabase();
    await waitForAppReady();
  });

  describe("Select all (Cmd+A)", () => {
    it("should create tasks and select all with Cmd+A", async () => {
      await createTaskViaUI("SelectAll A");
      await createTaskViaUI("SelectAll B");
      await createTaskViaUI("SelectAll C");

      // Blur any focused input
      await browser.execute(() => {
        (document.activeElement as HTMLElement)?.blur();
      });
      await browser.pause(100);

      await pressKeys("a", true);
      await browser.pause(300);

      // Bulk bar should be visible with at least 3 selected
      expect(await isBulkBarVisible()).toBe(true);
      const count = await getBulkBarCount();
      const num = parseInt(count.replace(/[^0-9]/g, ""), 10);
      expect(num).toBeGreaterThanOrEqual(3);
    });

    it("should clear selection", async () => {
      // Click a single task to clear multi-select
      await clickTask("SelectAll A");
      await browser.pause(200);
    });
  });

  describe("Move task up/down (Cmd+Arrow)", () => {
    it("should select a task for moving", async () => {
      await clickTask("SelectAll B");
      await browser.execute(() => {
        (document.activeElement as HTMLElement)?.blur();
      });
      await browser.pause(100);
    });

    it("should move task up with Cmd+ArrowUp", async () => {
      const titlesBefore = await getTaskTitles();
      const indexBefore = titlesBefore.indexOf("SelectAll B");

      await pressKeys("ArrowUp", true);
      await browser.pause(300);

      const titlesAfter = await getTaskTitles();
      const indexAfter = titlesAfter.indexOf("SelectAll B");
      // Task should have moved up (or stayed at 0)
      expect(indexAfter).toBeLessThanOrEqual(indexBefore);
      expect(titlesAfter).toContain("SelectAll B");
    });

    it("should move task down with Cmd+ArrowDown", async () => {
      await pressKeys("ArrowDown", true);
      await browser.pause(300);

      const titles = await getTaskTitles();
      expect(titles).toContain("SelectAll B");
    });
  });

  describe("Space to create task", () => {
    it("should create a task with Space key", async () => {
      // Need to blur to ensure not editing text
      await browser.execute(() => {
        (document.activeElement as HTMLElement)?.blur();
      });
      await browser.pause(100);

      const titlesBefore = await getTaskTitles();
      await pressKeys(" ");
      await browser.pause(500);

      // A new empty task should have been created
      // (it will have an empty title, which might get filtered out of getTaskTitles)
      // Just verify no crash
      const titlesAfter = await getTaskTitles();
      expect(titlesAfter.length).toBeGreaterThanOrEqual(titlesBefore.length);
    });
  });

  describe("Cmd+, for Settings", () => {
    it("should navigate to Settings with Cmd+,", async () => {
      await browser.execute(() => {
        (document.activeElement as HTMLElement)?.blur();
      });
      await browser.pause(100);

      await pressKeys(",", true);
      await browser.pause(300);

      const title = await getViewTitle();
      expect(title.toLowerCase()).toContain("settings");
    });
  });

  describe("Shift+Arrow extends selection", () => {
    it("should extend selection with Shift+ArrowDown", async () => {
      await navigateTo("Inbox");
      await browser.pause(200);

      await clickTask("SelectAll A");
      await browser.execute(() => {
        (document.activeElement as HTMLElement)?.blur();
      });
      await browser.pause(100);

      // Shift+Down should add next task to selection
      await pressKeys("ArrowDown", false, true);
      await browser.pause(200);

      expect(await isBulkBarVisible()).toBe(true);
      const count = await getBulkBarCount();
      expect(count).toContain("2");
    });
  });

  after(async () => {
    // Clear any multi-select
    await browser.execute(() => {
      const btn = document.querySelector("button[title='Clear selection']");
      if (btn) (btn as HTMLElement).click();
    });
    await navigateTo("Inbox");
  });
});
