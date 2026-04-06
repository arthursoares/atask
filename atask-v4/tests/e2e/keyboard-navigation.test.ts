import {
  waitForAppReady,
  resetDatabase,
  navigateTo,
  createTaskViaUI,
  getTaskTitles,
  clickTask,
  isDetailPanelVisible,
  getDetailTitle,
  pressKeys,
  isTaskCompleted,
  elementExists,
} from "./helpers";

describe("Keyboard Navigation & Shortcuts", () => {
  before(async () => {
    await waitForAppReady();
    await navigateTo("Inbox");
  });

  beforeEach(async () => {
    await resetDatabase();
    await waitForAppReady();
  });

  describe("Arrow key navigation", () => {
    it("should create tasks for navigation", async () => {
      await createTaskViaUI("Nav Task 1");
      await createTaskViaUI("Nav Task 2");
      await createTaskViaUI("Nav Task 3");
      const titles = await getTaskTitles();
      expect(titles.length).toBeGreaterThanOrEqual(3);
    });

    it("should select first task", async () => {
      await clickTask("Nav Task 1");
      expect(await isDetailPanelVisible()).toBe(true);
      expect(await getDetailTitle()).toBe("Nav Task 1");
    });

    it("should navigate down with Arrow Down", async () => {
      // Blur any focused input first
      await browser.execute(() => {
        (document.activeElement as HTMLElement)?.blur();
      });
      await browser.pause(100);
      await pressKeys("ArrowDown");
      await browser.pause(200);

      // Should now have "Nav Task 2" selected
      const selected = await browser.execute(() => {
        const items = document.querySelectorAll(".task-item.selected");
        if (items.length > 0) {
          const title = items[0].querySelector(".task-title");
          return title?.textContent ?? "";
        }
        return "";
      });
      expect(selected).toBe("Nav Task 2");
    });

    it("should navigate down with J key", async () => {
      await browser.execute(() => {
        (document.activeElement as HTMLElement)?.blur();
      });
      await browser.pause(100);
      await pressKeys("j");
      await browser.pause(200);

      const selected = await browser.execute(() => {
        const items = document.querySelectorAll(".task-item.selected");
        if (items.length > 0) {
          const title = items[0].querySelector(".task-title");
          return title?.textContent ?? "";
        }
        return "";
      });
      expect(selected).toBe("Nav Task 3");
    });

    it("should navigate up with Arrow Up", async () => {
      await browser.execute(() => {
        (document.activeElement as HTMLElement)?.blur();
      });
      await browser.pause(100);
      await pressKeys("ArrowUp");
      await browser.pause(200);

      const selected = await browser.execute(() => {
        const items = document.querySelectorAll(".task-item.selected");
        if (items.length > 0) {
          const title = items[0].querySelector(".task-title");
          return title?.textContent ?? "";
        }
        return "";
      });
      expect(selected).toBe("Nav Task 2");
    });

    it("should navigate up with K key", async () => {
      await browser.execute(() => {
        (document.activeElement as HTMLElement)?.blur();
      });
      await browser.pause(100);
      await pressKeys("k");
      await browser.pause(200);

      const selected = await browser.execute(() => {
        const items = document.querySelectorAll(".task-item.selected");
        if (items.length > 0) {
          const title = items[0].querySelector(".task-title");
          return title?.textContent ?? "";
        }
        return "";
      });
      expect(selected).toBe("Nav Task 1");
    });
  });

  describe("Task action shortcuts", () => {
    it("should duplicate a task with Cmd+D", async () => {
      await clickTask("Nav Task 1");
      await browser.execute(() => {
        (document.activeElement as HTMLElement)?.blur();
      });
      await browser.pause(100);
      await pressKeys("d", true);
      await browser.pause(500);

      const titles = await getTaskTitles();
      const navCount = titles.filter((t) => t === "Nav Task 1").length;
      expect(navCount).toBeGreaterThanOrEqual(2);
    });

    // TODO: synthetic KeyboardEvent with shift+meta doesn't propagate reliably in WebDriver
    it.skip("should complete a task with Shift+Cmd+C", async () => {
      await clickTask("Nav Task 3");
      await browser.execute(() => {
        (document.activeElement as HTMLElement)?.blur();
      });
      await browser.pause(100);
      await pressKeys("c", true, true);
      await browser.pause(300);

      expect(await isTaskCompleted("Nav Task 3")).toBe(true);
    });

    it("should schedule for Evening with Cmd+E", async () => {
      await clickTask("Nav Task 2");
      await browser.execute(() => {
        (document.activeElement as HTMLElement)?.blur();
      });
      await browser.pause(100);
      await pressKeys("e", true);
      await browser.pause(300);

      // Should disappear from Inbox (moved to Today Evening)
      const inboxTitles = await getTaskTitles();
      expect(inboxTitles).not.toContain("Nav Task 2");

      // Verify in Today view
      await navigateTo("Today");
      await browser.pause(300);
      const todayTitles = await getTaskTitles();
      expect(todayTitles).toContain("Nav Task 2");
    });
  });

  describe("Enter key opens detail panel", () => {
    // TODO: synthetic KeyboardEvent Enter doesn't trigger useKeyboard's handler in WebDriver context
    it.skip("should open detail panel with Enter key", async () => {
      await navigateTo("Inbox");
      await browser.pause(200);

      // Click a task first, then press Enter
      const titles = await getTaskTitles();
      if (titles.length > 0) {
        await clickTask(titles[0]);
        await pressKeys("Escape"); // close detail panel
        await browser.pause(200);

        // Now press Enter to reopen
        await browser.execute(() => {
          (document.activeElement as HTMLElement)?.blur();
        });
        await pressKeys("Enter");
        await browser.pause(300);

        // Detail panel OR inline editor should be visible
        const hasDetail = await isDetailPanelVisible();
        const hasEditor = await elementExists(".task-item.editing");
        expect(hasDetail || hasEditor).toBe(true);
      }
    });
  });

  after(async () => {
    await pressKeys("Escape");
    await navigateTo("Inbox");
  });
});
