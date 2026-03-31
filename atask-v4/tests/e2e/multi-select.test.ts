import {
  waitForAppReady,
  navigateTo,
  createTaskViaUI,
  getTaskTitles,
  countElements,
  elementExists,
} from "./helpers";

describe("Multi-Select & Bulk Operations", () => {
  before(async () => {
    await waitForAppReady();
    await navigateTo("Inbox");
    await createTaskViaUI("Multi A");
    await createTaskViaUI("Multi B");
    await createTaskViaUI("Multi C");
  });

  it("should Cmd+click to multi-select tasks", async () => {
    // Click first task
    await browser.execute(() => {
      const items = document.querySelectorAll(".task-item");
      if (items[0]) (items[0] as HTMLElement).click();
    });
    await browser.pause(200);

    // Cmd+click second task
    await browser.execute(() => {
      const items = document.querySelectorAll(".task-item");
      if (items[1]) {
        items[1].dispatchEvent(
          new MouseEvent("click", { metaKey: true, bubbles: true }),
        );
      }
    });
    await browser.pause(200);

    const selectedCount = await countElements(".task-item.selected");
    expect(selectedCount).toBeGreaterThanOrEqual(2);
  });

  it("should show bulk action bar", async () => {
    expect(await elementExists(".bulk-bar-btn")).toBe(true);
  });

  it("should clear selection with ✕", async () => {
    await browser.execute(() => {
      const btn = document.querySelector("button[title='Clear selection']");
      if (btn) (btn as HTMLElement).click();
    });
    await browser.pause(300);

    const selectedCount = await countElements(".task-item.selected");
    expect(selectedCount).toBeLessThanOrEqual(1);
  });
});
