import {
  waitForAppReady,
  navigateTo,
  createTaskViaUI,
  getTaskTitles,
  cmdClickTask,
  isBulkBarVisible,
  getBulkBarCount,
  clickBulkAction,
  isTaskCompleted,
  pressKeys,
} from "./helpers";

describe("Bulk Actions", () => {
  before(async () => {
    await waitForAppReady();
    await navigateTo("Inbox");
  });

  describe("Multi-select and bulk bar", () => {
    it("should create multiple tasks for bulk testing", async () => {
      await createTaskViaUI("Bulk A");
      await createTaskViaUI("Bulk B");
      await createTaskViaUI("Bulk C");
      await createTaskViaUI("Bulk D");
      await createTaskViaUI("Bulk E");
      const titles = await getTaskTitles();
      expect(titles).toContain("Bulk A");
      expect(titles).toContain("Bulk E");
    });

    it("should select multiple tasks with Cmd+click", async () => {
      await cmdClickTask("Bulk A");
      await cmdClickTask("Bulk B");
      await browser.pause(200);

      expect(await isBulkBarVisible()).toBe(true);
      const count = await getBulkBarCount();
      expect(count).toContain("2");
    });

    it("should add more tasks to selection", async () => {
      await cmdClickTask("Bulk C");
      const count = await getBulkBarCount();
      expect(count).toContain("3");
    });

    it("should deselect a task with Cmd+click", async () => {
      await cmdClickTask("Bulk A");
      const count = await getBulkBarCount();
      expect(count).toContain("2");
    });

    it("should clear selection via bulk bar X button", async () => {
      await clickBulkAction("\u2715"); // X button
      await browser.pause(200);
      expect(await isBulkBarVisible()).toBe(false);
    });
  });

  describe("Bulk complete", () => {
    it("should select and bulk complete tasks", async () => {
      await cmdClickTask("Bulk A");
      await cmdClickTask("Bulk B");
      await clickBulkAction("Complete");
      await browser.pause(300);

      expect(await isTaskCompleted("Bulk A")).toBe(true);
      expect(await isTaskCompleted("Bulk B")).toBe(true);
      expect(await isBulkBarVisible()).toBe(false);
    });
  });

  describe("Bulk schedule to Today", () => {
    it("should select and schedule tasks for Today", async () => {
      await cmdClickTask("Bulk C");
      await cmdClickTask("Bulk D");
      await clickBulkAction("Today");
      await browser.pause(300);

      // Tasks should disappear from Inbox
      const inboxTitles = await getTaskTitles();
      expect(inboxTitles).not.toContain("Bulk C");
      expect(inboxTitles).not.toContain("Bulk D");
    });

    it("should show scheduled tasks in Today view", async () => {
      await navigateTo("Today");
      await browser.pause(500);
      const todayTitles = await getTaskTitles();
      expect(todayTitles).toContain("Bulk C");
      expect(todayTitles).toContain("Bulk D");
    });
  });

  describe("Bulk schedule to Someday", () => {
    it("should move Today tasks to Someday", async () => {
      // Still on Today view
      await cmdClickTask("Bulk C");
      await cmdClickTask("Bulk D");
      await clickBulkAction("Someday");
      await browser.pause(300);

      await navigateTo("Someday");
      await browser.pause(300);
      const somedayTitles = await getTaskTitles();
      expect(somedayTitles).toContain("Bulk C");
      expect(somedayTitles).toContain("Bulk D");
    });
  });

  describe("Bulk delete", () => {
    it("should select and delete a task", async () => {
      await navigateTo("Inbox");
      await browser.pause(200);

      const titles = await getTaskTitles();
      if (titles.includes("Bulk E")) {
        await cmdClickTask("Bulk E");
        await clickBulkAction("Delete");
        await browser.pause(300);

        const after = await getTaskTitles();
        expect(after).not.toContain("Bulk E");
      }
    });
  });

  after(async () => {
    await navigateTo("Inbox");
  });
});
