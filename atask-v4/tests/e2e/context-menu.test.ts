import {
  waitForAppReady,
  navigateTo,
  createTaskViaUI,
  getTaskTitles,
  rightClickTask,
  clickContextMenuItem,
  isContextMenuVisible,
  isTaskCompleted,
  pressKeys,
} from "./helpers";

describe("Context Menu", () => {
  before(async () => {
    await waitForAppReady();
    await navigateTo("Inbox");
  });

  it("should create tasks for context menu testing", async () => {
    await createTaskViaUI("CTX Complete");
    await createTaskViaUI("CTX Today");
    await createTaskViaUI("CTX Someday");
    await createTaskViaUI("CTX Duplicate");
    await createTaskViaUI("CTX Delete");
    const titles = await getTaskTitles();
    expect(titles).toContain("CTX Complete");
    expect(titles).toContain("CTX Today");
    expect(titles).toContain("CTX Duplicate");
    expect(titles).toContain("CTX Delete");
  });

  it("should open context menu on right-click", async () => {
    await rightClickTask("CTX Complete");
    expect(await isContextMenuVisible()).toBe(true);
    // Dismiss it
    await browser.execute(() => {
      document.dispatchEvent(new MouseEvent("mousedown", { bubbles: true }));
    });
    await browser.pause(200);
  });

  it("should complete a task via context menu", async () => {
    await rightClickTask("CTX Complete");
    await clickContextMenuItem("Complete");
    await browser.pause(300);
    expect(await isTaskCompleted("CTX Complete")).toBe(true);
  });

  it("should delete a task via context menu", async () => {
    await rightClickTask("CTX Delete");
    await clickContextMenuItem("Delete");
    await browser.pause(300);
    const titles = await getTaskTitles();
    expect(titles).not.toContain("CTX Delete");
  });

  it("should duplicate a task via context menu", async () => {
    await rightClickTask("CTX Duplicate");
    await clickContextMenuItem("Duplicate");
    await browser.pause(500);
    const titles = await getTaskTitles();
    const dupeCount = titles.filter((t) => t.includes("CTX Duplicate")).length;
    expect(dupeCount).toBeGreaterThanOrEqual(2);
  });

  it("should schedule a task for Today via context menu", async () => {
    await rightClickTask("CTX Today");
    await clickContextMenuItem("Today");
    await browser.pause(300);

    // Task should leave Inbox
    const inboxTitles = await getTaskTitles();
    expect(inboxTitles).not.toContain("CTX Today");
  });

  it("should verify scheduled task appears in Today view", async () => {
    await navigateTo("Today");
    await browser.pause(500);
    const todayTitles = await getTaskTitles();
    expect(todayTitles).toContain("CTX Today");
  });

  it("should schedule a task for Someday via context menu", async () => {
    await navigateTo("Inbox");
    await browser.pause(300);
    await rightClickTask("CTX Someday");
    await clickContextMenuItem("Someday");
    await browser.pause(300);

    const inboxTitles = await getTaskTitles();
    expect(inboxTitles).not.toContain("CTX Someday");

    await navigateTo("Someday");
    await browser.pause(300);
    const somedayTitles = await getTaskTitles();
    expect(somedayTitles).toContain("CTX Someday");
  });

  after(async () => {
    await navigateTo("Inbox");
  });
});
