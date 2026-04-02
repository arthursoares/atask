import {
  waitForAppReady,
  navigateTo,
  createTaskViaUI,
  getTaskTitles,
} from "./helpers";

async function beginTaskDrag(title: string) {
  const started = await browser.execute((taskTitle: string) => {
    const items = Array.from(document.querySelectorAll(".task-item"));
    const dragItem = items.find((item) => {
      const titleEl = item.querySelector(".task-title");
      return titleEl?.textContent === taskTitle;
    }) as HTMLElement | undefined;

    if (!dragItem) return false;

    const dataTransfer = new DataTransfer();
    (window as typeof window & { __wdioTaskDragTransfer?: DataTransfer }).__wdioTaskDragTransfer = dataTransfer;

    dragItem.dispatchEvent(
      new DragEvent("dragstart", { bubbles: true, dataTransfer }),
    );

    return true;
  }, title);

  expect(started).toBe(true);
}

async function endTaskDrag(title: string) {
  await browser.execute((taskTitle: string) => {
    const items = Array.from(document.querySelectorAll(".task-item"));
    const dragItem = items.find((item) => {
      const titleEl = item.querySelector(".task-title");
      return titleEl?.textContent === taskTitle;
    }) as HTMLElement | undefined;

    dragItem?.dispatchEvent(new DragEvent("dragend", { bubbles: true }));
    delete (window as typeof window & { __wdioTaskDragTransfer?: DataTransfer }).__wdioTaskDragTransfer;
  }, title);
}

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

  it("should have draggable attribute on task rows", async () => {
    const draggableCount = await browser.execute(() => {
      const items = document.querySelectorAll(".task-item[draggable='true']");
      return items.length;
    });
    expect(draggableCount).toBeGreaterThanOrEqual(3);
  });

  it("should render slot drop zones only during an active drag", async () => {
    const beforeCount = await browser.execute(() => {
      return document.querySelectorAll(".task-drop-zone").length;
    });
    expect(beforeCount).toBe(0);

    const taskCount = await browser.execute(() => {
      return document.querySelectorAll(".task-item").length;
    });

    await beginTaskDrag("Drag C");

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

    await endTaskDrag("Drag C");

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

  it("should reorder tasks by dropping onto a slot drop zone", async () => {
    const dropIndex = await browser.execute(() => {
      const items = Array.from(document.querySelectorAll(".task-item"));
      return items.findIndex((item) => {
        const title = item.querySelector(".task-title");
        return title?.textContent === "Drag A";
      });
    });

    expect(dropIndex).toBeGreaterThanOrEqual(0);

    await beginTaskDrag("Drag C");

    await browser.waitUntil(
      async () => {
        const zoneCount = await browser.execute(() => {
          return document.querySelectorAll(".task-drop-zone").length;
        });
        return zoneCount > 0;
      },
      { timeout: 3000, timeoutMsg: "Task drop zones did not appear for reorder" },
    );

    await browser.execute((index: number) => {
      const transfer = (window as typeof window & { __wdioTaskDragTransfer?: DataTransfer }).__wdioTaskDragTransfer;
      if (!transfer) return;

      const dropZones = Array.from(document.querySelectorAll(".task-drop-zone"));
      const dropZone = dropZones[index] as HTMLElement | undefined;
      dropZone?.dispatchEvent(
        new DragEvent("dragover", { bubbles: true, dataTransfer: transfer }),
      );
    }, dropIndex);

    await browser.waitUntil(
      async () => {
        return browser.execute((index: number) => {
          const dropZones = Array.from(document.querySelectorAll(".task-drop-zone"));
          const dropZone = dropZones[index] as HTMLElement | undefined;
          return dropZone?.querySelector(".task-drop-slot") !== null;
        }, dropIndex);
      },
      { timeout: 3000, timeoutMsg: "Task drop slot did not become visible during dragover" },
    );

    await browser.execute((index: number) => {
      const transfer = (window as typeof window & { __wdioTaskDragTransfer?: DataTransfer }).__wdioTaskDragTransfer;
      if (!transfer) return;

      const dropZones = Array.from(document.querySelectorAll(".task-drop-zone"));
      const dropZone = dropZones[index] as HTMLElement | undefined;
      dropZone?.dispatchEvent(
        new DragEvent("drop", { bubbles: true, dataTransfer: transfer }),
      );
    }, dropIndex);

    await endTaskDrag("Drag C");

    await browser.waitUntil(
      async () => {
        const zoneCount = await browser.execute(() => {
          return document.querySelectorAll(".task-drop-zone").length;
        });
        return zoneCount === 0;
      },
      { timeout: 3000, timeoutMsg: "Task drop zones did not clear after drop" },
    );

    const titles = await getTaskTitles();

    const dragAIndex = titles.indexOf("Drag A");
    const dragCIndex = titles.indexOf("Drag C");
    expect(dragAIndex).toBeGreaterThanOrEqual(0);
    expect(dragCIndex).toBeGreaterThanOrEqual(0);
    expect(dragCIndex).toBeLessThan(dragAIndex);
  });

  after(async () => {
    await navigateTo("Inbox");
  });
});
