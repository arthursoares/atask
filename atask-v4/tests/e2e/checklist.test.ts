import {
  waitForAppReady,
  resetDatabase,
  navigateTo,
  createTaskViaUI,
  clickTask,
  isDetailPanelVisible,
  addChecklistItem,
  getChecklistItems,
  toggleChecklistItem,
  isChecklistItemDone,
  deleteChecklistItem,
  pressKeys,
} from "./helpers";

describe("Checklist", () => {
  before(async () => {
    await waitForAppReady();
    await navigateTo("Inbox");
  });

  beforeEach(async () => {
    await resetDatabase();
    await waitForAppReady();
  });

  it("should create a task and open detail panel", async () => {
    await createTaskViaUI("Checklist Host Task");
    await clickTask("Checklist Host Task");
    expect(await isDetailPanelVisible()).toBe(true);
  });

  it("should add a checklist item", async () => {
    await addChecklistItem("Step 1: Plan");
    const items = await getChecklistItems();
    expect(items).toContain("Step 1: Plan");
  });

  it("should add multiple checklist items", async () => {
    await addChecklistItem("Step 2: Build");
    await addChecklistItem("Step 3: Test");
    const items = await getChecklistItems();
    expect(items).toContain("Step 1: Plan");
    expect(items).toContain("Step 2: Build");
    expect(items).toContain("Step 3: Test");
    expect(items.length).toBe(3);
  });

  it("should toggle a checklist item to done", async () => {
    await toggleChecklistItem("Step 1: Plan");
    expect(await isChecklistItemDone("Step 1: Plan")).toBe(true);
  });

  it("should toggle a done item back to undone", async () => {
    await toggleChecklistItem("Step 1: Plan");
    expect(await isChecklistItemDone("Step 1: Plan")).toBe(false);
  });

  it("should delete a checklist item", async () => {
    await deleteChecklistItem("Step 2: Build");
    await browser.pause(300);
    const items = await getChecklistItems();
    expect(items).not.toContain("Step 2: Build");
    expect(items.length).toBe(2);
  });

  after(async () => {
    await pressKeys("Escape");
    await navigateTo("Inbox");
  });
});
