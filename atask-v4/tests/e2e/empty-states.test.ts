import {
  waitForAppReady,
  navigateTo,
  elementExists,
} from "./helpers";

describe("Empty States", () => {
  before(async () => {
    await waitForAppReady();
  });

  it("should show empty state in Inbox when fresh", async () => {
    await navigateTo("Inbox");
    await browser.pause(200);

    // Either has tasks or shows empty state
    const taskCount = await browser.execute(() => {
      return document.querySelectorAll(".task-item").length;
    });
    const hasEmptyState = await elementExists(".empty-state");

    // One of these must be true
    expect(taskCount > 0 || hasEmptyState).toBe(true);
  });

  it("should show empty state text in Today view", async () => {
    await navigateTo("Today");
    await browser.pause(200);

    const taskCount = await browser.execute(() => {
      return document.querySelectorAll(".task-item").length;
    });

    if (taskCount === 0) {
      const hasEmptyState = await elementExists(".empty-state");
      expect(hasEmptyState).toBe(true);

      const emptyText = await browser.execute(() => {
        const el = document.querySelector(".empty-state-text");
        return el?.textContent ?? "";
      });
      expect(emptyText.length).toBeGreaterThan(0);
    }
  });

  it("should show empty state in Upcoming view", async () => {
    await navigateTo("Upcoming");
    await browser.pause(200);

    const taskCount = await browser.execute(() => {
      return document.querySelectorAll(".task-item").length;
    });

    if (taskCount === 0) {
      const hasEmptyState = await elementExists(".empty-state");
      expect(hasEmptyState).toBe(true);
    }
  });

  it("should show empty state in Someday view", async () => {
    await navigateTo("Someday");
    await browser.pause(200);

    const taskCount = await browser.execute(() => {
      return document.querySelectorAll(".task-item").length;
    });

    if (taskCount === 0) {
      const hasEmptyState = await elementExists(".empty-state");
      expect(hasEmptyState).toBe(true);
    }
  });

  it("should show empty state in Logbook view", async () => {
    await navigateTo("Logbook");
    await browser.pause(200);

    const taskCount = await browser.execute(() => {
      return document.querySelectorAll(".task-item, .logbook-row").length;
    });

    if (taskCount === 0) {
      const hasEmptyState = await elementExists(".empty-state");
      expect(hasEmptyState).toBe(true);
    }
  });

  it("should show NewTaskRow in all editable views", async () => {
    for (const view of ["Inbox", "Today", "Someday"]) {
      await navigateTo(view);
      await browser.pause(200);

      const hasNewTask = await elementExists(".new-task-inline");
      expect(hasNewTask).toBe(true);
    }
  });

  after(async () => {
    await navigateTo("Inbox");
  });
});
