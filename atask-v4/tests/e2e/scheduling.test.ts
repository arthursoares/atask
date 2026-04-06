import {
  waitForAppReady,
  resetDatabase,
  navigateTo,
  createTaskViaUI,
  getTaskTitles,
  clickTask,
  isDetailPanelVisible,
  clickDetailField,
  isWhenPickerVisible,
  clickWhenOption,
  clickWhenClear,
  getDetailFieldValue,
  pressKeys,
  clickTriageAction,
} from "./helpers";

describe("Scheduling", () => {
  before(async () => {
    await waitForAppReady();
    await navigateTo("Inbox");
  });

  beforeEach(async () => {
    await resetDatabase();
    await waitForAppReady();
  });

  describe("Schedule via Detail Panel WhenPicker", () => {
    it("should create a task in Inbox", async () => {
      await createTaskViaUI("Schedule Test Task");
      const titles = await getTaskTitles();
      expect(titles).toContain("Schedule Test Task");
    });

    it("should open WhenPicker from detail panel", async () => {
      await clickTask("Schedule Test Task");
      expect(await isDetailPanelVisible()).toBe(true);

      await clickDetailField("Schedule");
      expect(await isWhenPickerVisible()).toBe(true);
    });

    it("should schedule task for Today via WhenPicker", async () => {
      await clickWhenOption("Today");
      await browser.pause(500);

      // Verify the schedule field updated
      const scheduleValue = await getDetailFieldValue("Schedule");
      expect(scheduleValue).toContain("Today");
    });

    it("should show the task in Today view", async () => {
      await browser.pause(500);
      await navigateTo("Today");
      await browser.pause(500);
      const titles = await getTaskTitles();
      expect(titles).toContain("Schedule Test Task");
    });

    it("should schedule task for Evening via WhenPicker", async () => {
      await clickTask("Schedule Test Task");
      await clickDetailField("Schedule");
      await clickWhenOption("This Evening");
      await browser.pause(300);

      const scheduleValue = await getDetailFieldValue("Schedule");
      expect(scheduleValue).toContain("Evening");
      await pressKeys("Escape");
    });

    it("should schedule task for Someday via WhenPicker", async () => {
      await clickTask("Schedule Test Task");
      await clickDetailField("Schedule");
      await clickWhenOption("Someday");
      await browser.pause(300);
      await pressKeys("Escape");
      await browser.pause(200);

      // Task should disappear from Today view
      const todayTitles = await getTaskTitles();
      expect(todayTitles).not.toContain("Schedule Test Task");
    });

    it("should show the task in Someday view", async () => {
      await navigateTo("Someday");
      await browser.pause(300);
      const titles = await getTaskTitles();
      expect(titles).toContain("Schedule Test Task");
    });

    it("should clear schedule to move back to Inbox", async () => {
      await clickTask("Schedule Test Task");
      await clickDetailField("Schedule");
      await clickWhenClear();
      await browser.pause(300);
      await pressKeys("Escape");
      await browser.pause(200);

      // Task should disappear from Someday
      const somedayTitles = await getTaskTitles();
      expect(somedayTitles).not.toContain("Schedule Test Task");

      // And appear in Inbox
      await navigateTo("Inbox");
      await browser.pause(300);
      const inboxTitles = await getTaskTitles();
      expect(inboxTitles).toContain("Schedule Test Task");
    });
  });

  describe("Schedule via Triage Actions (Inbox)", () => {
    it("should create a task for triage", async () => {
      await navigateTo("Inbox");
      await createTaskViaUI("Triage Test Task");
      const titles = await getTaskTitles();
      expect(titles).toContain("Triage Test Task");
    });

    it("should schedule task for Today via triage star button", async () => {
      await clickTriageAction("Triage Test Task", "today");
      await browser.pause(300);

      // Task should move out of inbox
      const inboxTitles = await getTaskTitles();
      expect(inboxTitles).not.toContain("Triage Test Task");

      // And appear in Today
      await navigateTo("Today");
      await browser.pause(300);
      const todayTitles = await getTaskTitles();
      expect(todayTitles).toContain("Triage Test Task");
    });
  });

  describe("Schedule via Keyboard Shortcuts", () => {
    it("should create and select a task", async () => {
      await navigateTo("Inbox");
      await createTaskViaUI("Keyboard Schedule Task");
      await clickTask("Keyboard Schedule Task");
      // Blur any focused input so keyboard handler works
      await browser.execute(() => {
        (document.activeElement as HTMLElement)?.blur();
      });
      await browser.pause(100);
    });

    it("should schedule for Today with Cmd+T", async () => {
      await pressKeys("t", true);
      await browser.pause(300);

      // Task should leave inbox
      const inboxTitles = await getTaskTitles();
      expect(inboxTitles).not.toContain("Keyboard Schedule Task");

      // Verify in Today view
      await navigateTo("Today");
      const todayTitles = await getTaskTitles();
      expect(todayTitles).toContain("Keyboard Schedule Task");
    });

    it("should schedule for Someday with Cmd+O", async () => {
      await clickTask("Keyboard Schedule Task");
      await browser.execute(() => {
        (document.activeElement as HTMLElement)?.blur();
      });
      await browser.pause(100);
      await pressKeys("o", true);
      await browser.pause(300);

      // Verify in Someday view
      await navigateTo("Someday");
      const somedayTitles = await getTaskTitles();
      expect(somedayTitles).toContain("Keyboard Schedule Task");
    });
  });

  after(async () => {
    await navigateTo("Inbox");
  });
});
