import {
  waitForAppReady,
  navigateTo,
  createTaskViaUI,
  getTaskTitles,
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

  it("should have draggable attribute on task rows", async () => {
    const draggableCount = await browser.execute(() => {
      const items = document.querySelectorAll(".task-item[draggable='true']");
      return items.length;
    });
    expect(draggableCount).toBeGreaterThanOrEqual(3);
  });

  it("should render slot drop zones only during an active drag", async () => {
    const dragState = await browser.execute(() => {
      const items = Array.from(document.querySelectorAll(".task-item"));
      const dragItem = items.find((item) => {
        const title = item.querySelector(".task-title");
        return title?.textContent === "Drag C";
      }) as HTMLElement | undefined;

      if (!dragItem) return null;

      const beforeCount = document.querySelectorAll(".task-drop-zone").length;
      const dataTransfer = new DataTransfer();
      dragItem.dispatchEvent(
        new DragEvent("dragstart", { bubbles: true, dataTransfer }),
      );

      const zoneCount = document.querySelectorAll(".task-drop-zone").length;
      const visibleSlotCount = document.querySelectorAll(".task-drop-slot").length;
      dragItem.dispatchEvent(new DragEvent("dragend", { bubbles: true }));
      const afterCount = document.querySelectorAll(".task-drop-zone").length;

      return {
        taskCount: items.length,
        beforeCount,
        zoneCount,
        visibleSlotCount,
        afterCount,
      };
    });

    expect(dragState).not.toBeNull();
    expect(dragState?.beforeCount).toBe(0);
    expect(dragState?.zoneCount).toBe(dragState!.taskCount + 1);
    expect(dragState?.visibleSlotCount).toBe(0);
    expect(dragState?.afterCount).toBe(0);
  });

  it("should reorder tasks by dropping onto a slot drop zone", async () => {
    const reordered = await browser.execute(() => {
      const items = Array.from(document.querySelectorAll(".task-item"));
      const dragItemIndex = items.findIndex((item) => {
        const title = item.querySelector(".task-title");
        return title?.textContent === "Drag C";
      });
      const dropIndex = items.findIndex((item) => {
        const title = item.querySelector(".task-title");
        return title?.textContent === "Drag A";
      });

      if (dragItemIndex === -1 || dropIndex === -1) return null;

      const dragItem = items[dragItemIndex] as HTMLElement;
      const dataTransfer = new DataTransfer();
      dragItem.dispatchEvent(
        new DragEvent("dragstart", { bubbles: true, dataTransfer }),
      );

      const dropZones = Array.from(document.querySelectorAll(".task-drop-zone"));
      const dropZone = dropZones[dropIndex] as HTMLElement | undefined;
      if (!dropZone) {
        dragItem.dispatchEvent(new DragEvent("dragend", { bubbles: true }));
        return null;
      }

      dropZone.dispatchEvent(
        new DragEvent("dragover", { bubbles: true, dataTransfer }),
      );

      const slotVisibleDuringDrag = dropZone.querySelector(".task-drop-slot") !== null;

      dropZone.dispatchEvent(
        new DragEvent("drop", { bubbles: true, dataTransfer }),
      );

      dragItem.dispatchEvent(new DragEvent("dragend", { bubbles: true }));

      const titles = Array.from(document.querySelectorAll(".task-title"))
        .map((el) => el.textContent ?? "")
        .filter(Boolean);

      return {
        slotVisibleDuringDrag,
        titles,
        remainingDropZones: document.querySelectorAll(".task-drop-zone").length,
      };
    });

    expect(reordered).not.toBeNull();
    expect(reordered?.slotVisibleDuringDrag).toBe(true);
    expect(reordered?.remainingDropZones).toBe(0);

    const dragAIndex = reordered!.titles.indexOf("Drag A");
    const dragCIndex = reordered!.titles.indexOf("Drag C");
    expect(dragAIndex).toBeGreaterThanOrEqual(0);
    expect(dragCIndex).toBeGreaterThanOrEqual(0);
    expect(dragCIndex).toBeLessThan(dragAIndex);
  });

  after(async () => {
    await navigateTo("Inbox");
  });
});
