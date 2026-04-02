import {
  waitForAppReady,
  navigateTo,
  createTaskViaUI,
  doubleClickTask,
  getTaskTitles,
  elementExists,
  getInlineEditorNotes,
  isInlineEditorOpen,
  setInlineEditorNotes,
  escapeInlineEditorNotes,
} from "./helpers";

describe("Inline Editor Pickers", () => {
  before(async () => {
    await waitForAppReady();
    await navigateTo("Inbox");
  });

  it("should create a task and open inline editor", async () => {
    await createTaskViaUI("Picker Test Task");
    await doubleClickTask("Picker Test Task");
    expect(await elementExists(".task-item.editing")).toBe(true);
  });

  it("should show attribute bar with picker buttons", async () => {
    expect(await elementExists(".attr-bar")).toBe(true);

    // Check for attribute pill buttons
    const pillCount = await browser.execute(() => {
      return document.querySelectorAll(".attr-pill").length;
    });
    expect(pillCount).toBeGreaterThan(0);
  });

  it("should have a When picker button", async () => {
    const hasWhenBtn = await browser.execute(() => {
      const pills = document.querySelectorAll(".attr-pill");
      for (const pill of pills) {
        if (pill.textContent?.includes("When")) return true;
      }
      return false;
    });
    expect(hasWhenBtn).toBe(true);
  });

  it("should have a Tag picker button", async () => {
    const hasTagBtn = await browser.execute(() => {
      const pills = document.querySelectorAll(".attr-pill");
      for (const pill of pills) {
        if (pill.textContent?.includes("Tag")) return true;
      }
      return false;
    });
    expect(hasTagBtn).toBe(true);
  });

  it("should have a Repeat picker button", async () => {
    const hasRepeatBtn = await browser.execute(() => {
      const pills = document.querySelectorAll(".attr-pill");
      for (const pill of pills) {
        if (pill.textContent?.includes("Repeat")) return true;
      }
      return false;
    });
    expect(hasRepeatBtn).toBe(true);
  });

  it("should have a Project picker button", async () => {
    const hasProjectBtn = await browser.execute(() => {
      const pills = document.querySelectorAll(".attr-pill");
      for (const pill of pills) {
        if (pill.textContent?.includes("Project")) return true;
      }
      return false;
    });
    expect(hasProjectBtn).toBe(true);
  });

  it("should open When picker from attribute bar", async () => {
    await browser.execute(() => {
      const pills = document.querySelectorAll(".attr-pill");
      for (const pill of pills) {
        if (pill.textContent?.includes("When")) {
          (pill as HTMLElement).click();
          return;
        }
      }
    });
    await browser.pause(300);

    const hasWhenPopover = await browser.execute(() => {
      return document.querySelector(".when-popover") !== null;
    });
    expect(hasWhenPopover).toBe(true);

    // Close it by clicking outside
    await browser.execute(() => {
      const editing = document.querySelector(".task-item.editing");
      if (editing) (editing as HTMLElement).click();
    });
    await browser.pause(200);
  });

  it("should open Repeat picker from attribute bar", async () => {
    await browser.execute(() => {
      const pills = document.querySelectorAll(".attr-pill");
      for (const pill of pills) {
        if (pill.textContent?.includes("Repeat")) {
          (pill as HTMLElement).click();
          return;
        }
      }
    });
    await browser.pause(300);

    // RepeatPicker renders a popover with "Repeat" header and "Daily", "Weekly" etc
    const hasRepeatPopover = await browser.execute(() => {
      const divs = document.querySelectorAll("div");
      for (const div of divs) {
        if (div.textContent?.includes("Daily") && div.textContent?.includes("Weekly")) {
          return true;
        }
      }
      return false;
    });
    expect(hasRepeatPopover).toBe(true);

    // Close by clicking outside
    await browser.execute(() => {
      const editing = document.querySelector(".task-item.editing");
      if (editing) (editing as HTMLElement).click();
    });
    await browser.pause(200);
  });

  it("should persist inline notes after closing with Escape from the notes field", async () => {
    const notes = "Inline note persisted through shared field";

    await setInlineEditorNotes(notes);
    expect(await getInlineEditorNotes()).toBe(notes);

    await escapeInlineEditorNotes();
    expect(await isInlineEditorOpen()).toBe(false);

    await doubleClickTask("Picker Test Task");
    expect(await getInlineEditorNotes()).toBe(notes);
  });

  it("should close inline editor with Escape and preserve task", async () => {
    // Close the inline editor
    await browser.execute(() => {
      const input = document.querySelector(".task-title-input") as HTMLInputElement;
      if (input) {
        input.dispatchEvent(
          new KeyboardEvent("keydown", { key: "Escape", code: "Escape", bubbles: true }),
        );
      }
    });
    await browser.pause(300);

    // Task should still exist
    const titles = await getTaskTitles();
    expect(titles).toContain("Picker Test Task");
  });

  after(async () => {
    await navigateTo("Inbox");
  });
});
