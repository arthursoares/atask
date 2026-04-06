import {
  waitForAppReady,
  resetDatabase,
  navigateTo,
  createTaskViaUI,
  getTaskTitles,
  clickCheckbox,
  isTaskCompleted,
  getLogbookTitles,
  isLogbookEntryCancelled,
  clickReopenInLogbook,
  rightClickTask,
  clickContextMenuItem,
  pressKeys,
} from "./helpers";

describe("Logbook", () => {
  before(async () => {
    await waitForAppReady();
    await navigateTo("Inbox");
  });

  beforeEach(async () => {
    await resetDatabase();
    await waitForAppReady();
  });

  it("should create tasks to complete and cancel", async () => {
    await createTaskViaUI("Logbook Complete Task");
    await createTaskViaUI("Logbook Cancel Task");
    await createTaskViaUI("Logbook Reopen Task");
    const titles = await getTaskTitles();
    expect(titles).toContain("Logbook Complete Task");
    expect(titles).toContain("Logbook Cancel Task");
    expect(titles).toContain("Logbook Reopen Task");
  });

  it("should complete a task via checkbox", async () => {
    await clickCheckbox("Logbook Complete Task");
    expect(await isTaskCompleted("Logbook Complete Task")).toBe(true);
  });

  it("should cancel a task via context menu", async () => {
    await rightClickTask("Logbook Cancel Task");
    await clickContextMenuItem("Cancel");
    await browser.pause(300);
  });

  it("should complete the reopen task", async () => {
    await clickCheckbox("Logbook Reopen Task");
    expect(await isTaskCompleted("Logbook Reopen Task")).toBe(true);
  });

  it("should show completed tasks in Logbook", async () => {
    await navigateTo("Logbook");
    await browser.pause(500);
    const titles = await getLogbookTitles();
    expect(titles).toContain("Logbook Complete Task");
    expect(titles).toContain("Logbook Reopen Task");
  });

  it("should show cancelled task in Logbook with cancelled indicator", async () => {
    const titles = await getLogbookTitles();
    expect(titles).toContain("Logbook Cancel Task");
    expect(await isLogbookEntryCancelled("Logbook Cancel Task")).toBe(true);
  });

  it("should reopen a task from Logbook", async () => {
    await clickReopenInLogbook("Logbook Reopen Task");
    await browser.pause(300);

    // Task should disappear from logbook
    const logbookTitles = await getLogbookTitles();
    expect(logbookTitles).not.toContain("Logbook Reopen Task");

    // And reappear in Inbox
    await navigateTo("Inbox");
    await browser.pause(300);
    const inboxTitles = await getTaskTitles();
    expect(inboxTitles).toContain("Logbook Reopen Task");
  });

  after(async () => {
    await navigateTo("Inbox");
  });
});
