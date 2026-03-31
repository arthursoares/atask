import {
  waitForAppReady,
  navigateTo,
  createTaskViaUI,
  getTaskTitles,
  clickTask,
  clickCheckbox,
  isTaskCompleted,
  isDetailPanelVisible,
  getDetailFieldValue,
  getLogbookTitles,
  clickReopenInLogbook,
  pressKeys,
} from "./helpers";

describe("Task Completion Flow", () => {
  before(async () => {
    await waitForAppReady();
    await navigateTo("Inbox");
  });

  describe("Complete and verify in Logbook", () => {
    it("should create and complete a task", async () => {
      await createTaskViaUI("Flow Complete Task");
      await clickCheckbox("Flow Complete Task");
      expect(await isTaskCompleted("Flow Complete Task")).toBe(true);
    });

    it("should still show completed task in Inbox (completed today)", async () => {
      // Tasks completed today should remain visible with strikethrough
      const titles = await getTaskTitles();
      expect(titles).toContain("Flow Complete Task");
    });

    it("should show in Logbook", async () => {
      await navigateTo("Logbook");
      await browser.pause(300);
      const titles = await getLogbookTitles();
      expect(titles).toContain("Flow Complete Task");
    });

    it("should reopen from Logbook and return to Inbox", async () => {
      await clickReopenInLogbook("Flow Complete Task");
      await browser.pause(300);

      await navigateTo("Inbox");
      await browser.pause(300);
      const titles = await getTaskTitles();
      expect(titles).toContain("Flow Complete Task");
      expect(await isTaskCompleted("Flow Complete Task")).toBe(false);
    });
  });

  describe("Complete task in Today view", () => {
    it("should schedule task for Today and complete it", async () => {
      await navigateTo("Inbox");
      await createTaskViaUI("Today Complete Task");
      await clickTask("Today Complete Task");
      await browser.execute(() => {
        (document.activeElement as HTMLElement)?.blur();
      });
      await browser.pause(100);

      // Schedule for Today
      await pressKeys("t", true);
      await browser.pause(300);

      // Navigate to Today
      await navigateTo("Today");
      await browser.pause(300);
      const titles = await getTaskTitles();
      expect(titles).toContain("Today Complete Task");

      // Complete it
      await clickCheckbox("Today Complete Task");
      expect(await isTaskCompleted("Today Complete Task")).toBe(true);
    });

    it("should show completed Today task in Logbook", async () => {
      await navigateTo("Logbook");
      await browser.pause(300);
      const titles = await getLogbookTitles();
      expect(titles).toContain("Today Complete Task");
    });
  });

  describe("Cancel task flow", () => {
    it("should create and cancel a task via context menu", async () => {
      await navigateTo("Inbox");
      await createTaskViaUI("Cancel Flow Task");

      // Right-click and cancel
      await browser.execute(() => {
        const items = document.querySelectorAll(".task-item");
        for (const item of items) {
          const title = item.querySelector(".task-title");
          if (title?.textContent === "Cancel Flow Task") {
            item.dispatchEvent(
              new MouseEvent("contextmenu", { bubbles: true, clientX: 300, clientY: 200 }),
            );
            return;
          }
        }
      });
      await browser.pause(200);

      // Click "Cancel" in context menu
      await browser.execute(() => {
        const allDivs = document.querySelectorAll("div");
        for (const div of allDivs) {
          const style = (div as HTMLElement).style;
          if (style.cursor === "pointer") {
            const spans = div.querySelectorAll("span");
            for (const span of spans) {
              if (span.textContent?.trim() === "Cancel") {
                (div as HTMLElement).click();
                return;
              }
            }
          }
        }
      });
      await browser.pause(300);
    });

    it("should show cancelled task in Logbook with indicator", async () => {
      await navigateTo("Logbook");
      await browser.pause(300);
      const titles = await getLogbookTitles();
      expect(titles).toContain("Cancel Flow Task");

      // Check for "Cancelled" pill
      const hasCancelledPill = await browser.execute(() => {
        const rows = document.querySelectorAll(".logbook-row");
        for (const row of rows) {
          const title = row.querySelector(".task-title");
          if (title?.textContent?.trim() === "Cancel Flow Task") {
            return row.textContent?.includes("Cancelled") ?? false;
          }
        }
        return false;
      });
      expect(hasCancelledPill).toBe(true);
    });
  });

  after(async () => {
    await navigateTo("Inbox");
  });
});
