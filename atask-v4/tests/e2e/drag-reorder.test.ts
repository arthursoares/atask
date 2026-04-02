import {
  waitForAppReady,
  navigateTo,
  createTaskViaUI,
  getTaskTitles,
  dragTaskByTitleToTaskByTitle,
  finishPointerDrag,
  getNativeTaskDragPayload,
  startPointerDragTaskByTitle,
  blurWindow,
} from "./helpers";

describe("Drag and Drop Reorder", () => {
  before(async () => {
    await waitForAppReady();
    await navigateTo("Inbox");
  });

  it("should create tasks for drag testing", async () => {
    await createTaskViaUI("Drag A");
    await createTaskViaUI("Drag B");
    await createTaskViaUI("Drag C");
    const titles = await getTaskTitles();
    expect(titles).toContain("Drag A");
    expect(titles).toContain("Drag B");
    expect(titles).toContain("Drag C");
  });

  it("should render slot drop zones only during an active pointer drag", async () => {
    const beforeCount = await browser.execute(() => {
      return document.querySelectorAll(".task-drop-zone").length;
    });
    expect(beforeCount).toBe(0);

    const taskCount = await browser.execute(() => {
      return document.querySelectorAll(".task-item").length;
    });

    await startPointerDragTaskByTitle("Drag C");

    await browser.waitUntil(
      async () => {
        const zoneCount = await browser.execute(() => {
          return document.querySelectorAll(".task-drop-zone").length;
        });
        return zoneCount === taskCount + 1;
      },
      { timeout: 3000, timeoutMsg: "Task drop zones did not appear during drag" },
    );

    const visibleSlotCount = await browser.execute(() => {
      return document.querySelectorAll(".task-drop-slot").length;
    });
    expect(visibleSlotCount).toBe(0);

    await finishPointerDrag();

    await browser.waitUntil(
      async () => {
        const afterCount = await browser.execute(() => {
          return document.querySelectorAll(".task-drop-zone").length;
        });
        return afterCount === 0;
      },
      { timeout: 3000, timeoutMsg: "Task drop zones did not clear after drag end" },
    );
  });

  it("should expose native task drag payloads for sidebar drops without starting pointer reorder", async () => {
    const dragPayload = await getNativeTaskDragPayload("Drag B");

    expect(dragPayload.draggable).toBe(true);
    expect(dragPayload.typeCalls).toContain("text/plain");
    expect(dragPayload.payload).toBeTruthy();
    expect(dragPayload.visibleDropZoneCount).toBe(0);
  });

  it("should reorder tasks with pointer dragging", async () => {
    await dragTaskByTitleToTaskByTitle("Drag C", "Drag A");

    const titles = await getTaskTitles();

    const dragAIndex = titles.indexOf("Drag A");
    const dragCIndex = titles.indexOf("Drag C");
    expect(dragAIndex).toBeGreaterThanOrEqual(0);
    expect(dragCIndex).toBeGreaterThanOrEqual(0);
    expect(dragCIndex).toBeLessThan(dragAIndex);
  });

  it("should cancel pointer reorder when the window blurs during mouse dragging", async () => {
    await startPointerDragTaskByTitle("Drag B");

    await browser.waitUntil(
      async () => {
        const zoneCount = await browser.execute(() => {
          return document.querySelectorAll(".task-drop-zone").length;
        });
        return zoneCount > 0;
      },
      { timeout: 3000, timeoutMsg: "Task drop zones did not appear during drag" },
    );

    await blurWindow();

    await browser.waitUntil(
      async () => {
        const zoneCount = await browser.execute(() => {
          return document.querySelectorAll(".task-drop-zone").length;
        });
        return zoneCount === 0;
      },
      { timeout: 3000, timeoutMsg: "Task drop zones did not clear after window blur" },
    );

    await finishPointerDrag();
  });

  after(async () => {
    await navigateTo("Inbox");
  });
});
