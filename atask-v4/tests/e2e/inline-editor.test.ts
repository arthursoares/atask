import {
  waitForAppReady,
  resetDatabase,
  navigateTo,
  createTaskViaUI,
  doubleClickTask,
  getTaskTitles,
  elementExists,
  pressKeys,
} from "./helpers";

describe("Inline Editor", () => {
  before(async () => {
    await waitForAppReady();
    await navigateTo("Inbox");
    await createTaskViaUI("Editor Test");
  });

  beforeEach(async () => {
    await resetDatabase();
    await waitForAppReady();
  });

  it("should open on double-click", async () => {
    await doubleClickTask("Editor Test");
    expect(await elementExists(".task-item.editing")).toBe(true);
  });

  it("should have the task title in the input", async () => {
    const value = await browser.execute(() => {
      const input = document.querySelector(".task-title-input") as HTMLInputElement;
      return input?.value ?? "";
    });
    expect(value).toBe("Editor Test");
  });

  it("should show attribute bar", async () => {
    expect(await elementExists(".attr-bar")).toBe(true);
    const addCount = await browser.execute(() => {
      return document.querySelectorAll(".attr-pill.attr-add").length;
    });
    expect(addCount).toBeGreaterThan(0);
  });

  it("should close with Escape", async () => {
    // Dispatch Escape on the title input (where the keydown handler lives)
    await browser.execute(() => {
      const input = document.querySelector(".task-title-input") as HTMLInputElement;
      if (input) {
        input.dispatchEvent(
          new KeyboardEvent("keydown", { key: "Escape", code: "Escape", bubbles: true }),
        );
      }
    });
    await browser.pause(300);
    expect(await elementExists(".task-item.editing")).toBe(false);
  });

  it("should delete task when title is emptied on close", async () => {
    await createTaskViaUI("Deletable");
    await doubleClickTask("Deletable");
    await browser.pause(200);

    // Clear the title
    await browser.execute(() => {
      const input = document.querySelector(".task-title-input") as HTMLInputElement;
      if (input) {
        input.value = "";
        input.dispatchEvent(new Event("input", { bubbles: true }));
        input.dispatchEvent(new Event("change", { bubbles: true }));
      }
    });
    await browser.pause(100);

    // Close the editor
    await pressKeys("Escape");
    await browser.pause(500);

    const titles = await getTaskTitles();
    expect(titles).not.toContain("Deletable");
  });
});
