import {
  waitForAppReady,
  navigateTo,
  createTaskViaUI,
  getTaskTitles,
  clickTask,
  isDetailPanelVisible,
  getDetailTitle,
  clickCheckbox,
  isTaskCompleted,
  pressKeys,
} from "./helpers";

describe("Task CRUD", () => {
  before(async () => {
    await waitForAppReady();
    await navigateTo("Inbox");
  });

  it("should create a task", async () => {
    await createTaskViaUI("E2E Alpha");
    const titles = await getTaskTitles();
    expect(titles).toContain("E2E Alpha");
  });

  it("should create a second task", async () => {
    await createTaskViaUI("E2E Beta");
    const titles = await getTaskTitles();
    expect(titles).toContain("E2E Beta");
    expect(titles).toContain("E2E Alpha");
  });

  it("should select a task and show detail panel", async () => {
    await clickTask("E2E Alpha");
    expect(await isDetailPanelVisible()).toBe(true);
    expect(await getDetailTitle()).toBe("E2E Alpha");
  });

  it("should complete a task via checkbox", async () => {
    await clickCheckbox("E2E Alpha");
    expect(await isTaskCompleted("E2E Alpha")).toBe(true);
  });

  // TODO: synthetic KeyboardEvent doesn't trigger useKeyboard's Backspace handler
  // because the WebDriver plugin's execute/sync context may not propagate to React's event system
  it.skip("should delete a task via Backspace", async () => {
    await clickTask("E2E Beta");
    await browser.pause(200);
    // Blur any focused input so isEditingText() returns false
    await browser.execute(() => {
      (document.activeElement as HTMLElement)?.blur();
    });
    await browser.pause(100);
    await pressKeys("Backspace");
    await browser.pause(500);
    const titles = await getTaskTitles();
    expect(titles).not.toContain("E2E Beta");
  });

  it("should close detail panel with Escape", async () => {
    const titles = await getTaskTitles();
    if (titles.length > 0) {
      await clickTask(titles[0]);
      expect(await isDetailPanelVisible()).toBe(true);
      await pressKeys("Escape");
      expect(await isDetailPanelVisible()).toBe(false);
    }
  });
});
