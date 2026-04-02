/**
 * E2E test helpers for atask v4 Tauri app.
 *
 * Uses browser.execute() with raw DOM queries to avoid
 * tauri-plugin-webdriver-automation's limited Node.contains support.
 */

// ---------------------------------------------------------------------------
// App lifecycle
// ---------------------------------------------------------------------------

/** Wait for the app to be fully loaded */
export async function waitForAppReady() {
  await browser.waitUntil(
    async () => {
      return browser.execute(() => document.querySelector(".sidebar") !== null);
    },
    { timeout: 15000, timeoutMsg: "App did not load within 15s" },
  );
}

// ---------------------------------------------------------------------------
// Navigation
// ---------------------------------------------------------------------------

/** Click a sidebar nav item by its text content */
export async function navigateTo(viewName: string) {
  const found = await browser.execute((name: string) => {
    const items = document.querySelectorAll(".sidebar-item");
    for (const item of items) {
      if (item.textContent?.includes(name)) {
        (item as HTMLElement).click();
        return true;
      }
    }
    return false;
  }, viewName);
  if (!found) {
    const labels = await browser.execute(() => {
      const items = document.querySelectorAll(".sidebar-item");
      return Array.from(items).map((el) => el.textContent?.trim() ?? "(empty)");
    });
    throw new Error(`Sidebar item "${viewName}" not found. Available: ${JSON.stringify(labels)}`);
  }
  await browser.pause(300);
}

/** Get the current view title from the toolbar */
export async function getViewTitle(): Promise<string> {
  return browser.execute(() => {
    const el = document.querySelector(".app-view-title");
    return el?.textContent ?? "";
  });
}

/** Get all sidebar item labels */
export async function getSidebarLabels(): Promise<string[]> {
  return browser.execute(() => {
    const items = document.querySelectorAll(".sidebar-item");
    return Array.from(items).map((el) => el.textContent?.trim() ?? "");
  });
}

// ---------------------------------------------------------------------------
// Task CRUD
// ---------------------------------------------------------------------------

/** Create a task using the NewTaskRow */
export async function createTaskViaUI(title: string) {
  // Click the "New Task" row
  await browser.execute(() => {
    const row = document.querySelector(".new-task-inline");
    if (row) (row as HTMLElement).click();
  });
  await browser.pause(300);

  // Type the title into the input -- must use React's native value setter
  await browser.execute((t: string) => {
    const input = document.querySelector(".new-task-inline input") as HTMLInputElement;
    if (input) {
      const nativeSet = Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value")?.set;
      if (nativeSet) {
        nativeSet.call(input, t);
        input.dispatchEvent(new Event("input", { bubbles: true }));
      }
    }
  }, title);
  await browser.pause(100);

  // Press Enter
  await browser.execute(() => {
    const input = document.querySelector(".new-task-inline input") as HTMLInputElement;
    if (input) {
      input.dispatchEvent(
        new KeyboardEvent("keydown", { key: "Enter", code: "Enter", bubbles: true }),
      );
    }
  });
  await browser.pause(500);

  // Press Escape to exit creation mode
  await browser.execute(() => {
    document.dispatchEvent(
      new KeyboardEvent("keydown", { key: "Escape", code: "Escape", bubbles: true }),
    );
  });
  await browser.pause(200);
}

/** Get all visible task titles in the current view */
export async function getTaskTitles(): Promise<string[]> {
  return browser.execute(() => {
    const items = document.querySelectorAll(".task-title");
    return Array.from(items).map((el) => el.textContent ?? "").filter(Boolean);
  });
}

export async function startPointerDragTaskByTitle(title: string) {
  const started = await browser.execute((taskTitle: string) => {
    const rows = Array.from(document.querySelectorAll(".task-item"));
    const source = rows.find((row) => {
      const titleEl = row.querySelector(".task-title");
      return titleEl?.textContent === taskTitle;
    }) as HTMLElement | undefined;
    if (!source) {
      return false;
    }

    const rect = source.getBoundingClientRect();
    const clientX = Math.round(rect.left + 16);
    const clientY = Math.round(rect.top + rect.height / 2);

    source.dispatchEvent(new MouseEvent("mousedown", {
      bubbles: true,
      cancelable: true,
      view: window,
      button: 0,
      buttons: 1,
      clientX,
      clientY,
    }));

    window.dispatchEvent(new MouseEvent("mousemove", {
      bubbles: true,
      cancelable: true,
      view: window,
      button: 0,
      buttons: 1,
      clientX,
      clientY: clientY + 8,
    }));

    return true;
  }, title);

  if (!started) {
    throw new Error(`Task "${title}" not found`);
  }
}

export async function getNativeTaskDragPayload(title: string): Promise<{
  draggable: boolean;
  payload: string | null;
  typeCalls: string[];
  visibleDropZoneCount: number;
}> {
  const result = await browser.execute((taskTitle: string) => {
    const rows = Array.from(document.querySelectorAll(".task-item"));
    const source = rows.find((row) => {
      const titleEl = row.querySelector(".task-title");
      return titleEl?.textContent === taskTitle;
    }) as HTMLElement | undefined;
    if (!source) {
      return null;
    }

    const payloadByType = new Map<string, string>();
    const typeCalls: string[] = [];
    const dataTransfer = {
      setData(type: string, value: string) {
        typeCalls.push(type);
        payloadByType.set(type, value);
      },
      getData(type: string) {
        return payloadByType.get(type) ?? "";
      },
      effectAllowed: "uninitialized",
    };

    source.dispatchEvent(new DragEvent("dragstart", {
      bubbles: true,
      cancelable: true,
      dataTransfer: dataTransfer as unknown as DataTransfer,
    }));

    return {
      draggable: source.draggable,
      payload: payloadByType.get("text/plain") ?? null,
      typeCalls,
      visibleDropZoneCount: document.querySelectorAll(".task-drop-zone").length,
    };
  }, title);

  if (!result) {
    throw new Error(`Task "${title}" not found`);
  }

  return result;
}

export async function finishPointerDrag() {
  await browser.execute(() => {
    window.dispatchEvent(new MouseEvent("mouseup", {
      bubbles: true,
      button: 0,
      buttons: 0,
    }));
  });
}

export async function dragTaskByTitleToTaskByTitle(sourceTitle: string, targetTitle: string) {
  const reordered = await browser.execute((sourceTaskTitle: string, targetTaskTitle: string) => {
    const rows = Array.from(document.querySelectorAll(".task-item"));
    const source = rows.find((row) => {
      const titleEl = row.querySelector(".task-title");
      return titleEl?.textContent === sourceTaskTitle;
    }) as HTMLElement | undefined;
    const target = rows.find((row) => {
      const titleEl = row.querySelector(".task-title");
      return titleEl?.textContent === targetTaskTitle;
    }) as HTMLElement | undefined;
    if (!source || !target) {
      return false;
    }

    const sourceRect = source.getBoundingClientRect();
    const targetRect = target.getBoundingClientRect();
    const sourceX = Math.round(sourceRect.left + 16);
    const sourceY = Math.round(sourceRect.top + sourceRect.height / 2);
    const targetY = Math.round(targetRect.top + 4);

    source.dispatchEvent(new MouseEvent("mousedown", {
      bubbles: true,
      button: 0,
      buttons: 1,
      clientX: sourceX,
      clientY: sourceY,
    }));

    window.dispatchEvent(new MouseEvent("mousemove", {
      bubbles: true,
      button: 0,
      buttons: 1,
      clientX: sourceX,
      clientY: targetY,
    }));

    window.dispatchEvent(new MouseEvent("mouseup", {
      bubbles: true,
      button: 0,
      buttons: 0,
      clientX: sourceX,
      clientY: targetY,
    }));

    return true;
  }, sourceTitle, targetTitle);

  if (!reordered) {
    throw new Error(`Unable to reorder "${sourceTitle}" to "${targetTitle}"`);
  }
}

export async function blurWindow() {
  await browser.execute(() => {
    window.dispatchEvent(new Event("blur"));
  });
}

export async function getSidebarAreaIds(): Promise<string[]> {
  return browser.execute(() => {
    return Array.from(document.querySelectorAll(".sidebar-group-label"))
      .map((item) => item.textContent?.trim() ?? "")
      .filter(Boolean);
  });
}

export async function startSidebarAreaDrag(sourceIndex: number, targetIndex: number) {
  const dragState = await browser.executeAsync((from: number, to: number, done: (result: Record<string, unknown>) => void) => {
    const areaItems = Array.from(
      document.querySelectorAll(".sidebar-group-label"),
    ) as HTMLElement[];
    const source = areaItems[from];
    const target = areaItems[to];
    if (!source || !target) {
      done({ ok: false, reason: "missing-area-item" });
      return;
    }

    const sourceRect = source.getBoundingClientRect();
    const targetRect = target.getBoundingClientRect();
    const sourceX = Math.round(sourceRect.left + 24);
    const sourceY = Math.round(sourceRect.top + sourceRect.height / 2);
    const targetY = from > to
      ? Math.round(targetRect.top + 6)
      : Math.round(targetRect.bottom - 6);

    source.dispatchEvent(new MouseEvent("mousedown", {
      bubbles: true,
      cancelable: true,
      view: window,
      button: 0,
      buttons: 1,
      clientX: sourceX,
      clientY: sourceY,
    }));

    window.dispatchEvent(new MouseEvent("mousemove", {
      bubbles: true,
      cancelable: true,
      view: window,
      button: 0,
      buttons: 1,
      clientX: sourceX,
      clientY: targetY,
    }));

    window.setTimeout(() => {
      const draggingItem = document.querySelector(".sidebar-group-label.sidebar-item-dragging");
      done({
        ok: true,
        activeId: draggingItem?.textContent?.trim() ?? null,
        hasDraggedClass: Boolean(draggingItem),
        visibleSlotCount: document.querySelectorAll(".sidebar-drop-slot").length,
      });
    }, 60);
  }, sourceIndex, targetIndex);

  if (!dragState?.ok) {
    throw new Error(`Unable to start sidebar area drag: ${JSON.stringify(dragState)}`);
  }

  return dragState as {
    activeId: string | null;
    hasDraggedClass: boolean;
    visibleSlotCount: number;
  };
}

export async function finishSidebarAreaDrag(sourceIndex: number, targetIndex: number) {
  const finished = await browser.executeAsync((from: number, to: number, done: (result: boolean) => void) => {
    const areaItems = Array.from(
      document.querySelectorAll(".sidebar-group-label"),
    ) as HTMLElement[];
    const source = areaItems[from];
    const target = areaItems[to];
    if (!source || !target) {
      done(false);
      return;
    }

    const sourceRect = source.getBoundingClientRect();
    const targetRect = target.getBoundingClientRect();
    const sourceX = Math.round(sourceRect.left + 24);
    const targetY = from > to
      ? Math.round(targetRect.top + 6)
      : Math.round(targetRect.bottom - 6);

    window.dispatchEvent(new MouseEvent("mousemove", {
      bubbles: true,
      cancelable: true,
      view: window,
      button: 0,
      buttons: 1,
      clientX: sourceX,
      clientY: targetY,
    }));

    window.dispatchEvent(new MouseEvent("mouseup", {
      bubbles: true,
      cancelable: true,
      view: window,
      button: 0,
      buttons: 0,
      clientX: sourceX,
      clientY: targetY,
    }));

    window.setTimeout(() => done(true), 60);
  }, sourceIndex, targetIndex);

  if (!finished) {
    throw new Error(`Unable to finish sidebar area drag from ${sourceIndex} to ${targetIndex}`);
  }
}

export async function dragSidebarArea(sourceIndex: number, targetIndex: number) {
  await startSidebarAreaDrag(sourceIndex, targetIndex);
  await finishSidebarAreaDrag(sourceIndex, targetIndex);
}

/** Click on a task row by its title */
export async function clickTask(title: string) {
  await browser.execute((t: string) => {
    const items = document.querySelectorAll(".task-item");
    for (const item of items) {
      const titleEl = item.querySelector(".task-title");
      if (titleEl?.textContent === t) {
        (item as HTMLElement).click();
        return;
      }
    }
    throw new Error(`Task "${t}" not found`);
  }, title);
  await browser.pause(200);
}

/** Cmd+click a task (for multi-select) */
export async function cmdClickTask(title: string) {
  await browser.execute((t: string) => {
    const items = document.querySelectorAll(".task-item");
    for (const item of items) {
      const titleEl = item.querySelector(".task-title");
      if (titleEl?.textContent === t) {
        (item as HTMLElement).dispatchEvent(
          new MouseEvent("click", { bubbles: true, metaKey: true }),
        );
        return;
      }
    }
    throw new Error(`Task "${t}" not found for cmd+click`);
  }, title);
  await browser.pause(200);
}

/** Double-click a task row to open inline editor */
export async function doubleClickTask(title: string) {
  await browser.execute((t: string) => {
    const items = document.querySelectorAll(".task-item");
    for (const item of items) {
      const titleEl = item.querySelector(".task-title");
      if (titleEl?.textContent === t) {
        (item as HTMLElement).dispatchEvent(
          new MouseEvent("dblclick", { bubbles: true }),
        );
        return;
      }
    }
    throw new Error(`Task "${t}" not found`);
  }, title);
  await browser.pause(300);
}

/** Click a task's checkbox */
export async function clickCheckbox(title: string) {
  await browser.execute((t: string) => {
    const items = document.querySelectorAll(".task-item");
    for (const item of items) {
      const titleEl = item.querySelector(".task-title");
      if (titleEl?.textContent === t) {
        const cb = item.querySelector(".checkbox");
        if (cb) (cb as HTMLElement).click();
        return;
      }
    }
  }, title);
  await browser.pause(500);
}

/** Check if a task has the "completed" class */
export async function isTaskCompleted(title: string): Promise<boolean> {
  return browser.execute((t: string) => {
    const items = document.querySelectorAll(".task-item");
    for (const item of items) {
      const titleEl = item.querySelector(".task-title");
      if (titleEl?.textContent === t) {
        return titleEl.classList.contains("completed");
      }
    }
    return false;
  }, title);
}

/** Count visible tasks in the current view */
export async function getTaskCount(): Promise<number> {
  return browser.execute(() => {
    return document.querySelectorAll(".task-item").length;
  });
}

// ---------------------------------------------------------------------------
// Keyboard
// ---------------------------------------------------------------------------

/** Send a keyboard shortcut */
export async function pressKeys(key: string, meta = false, shift = false) {
  await browser.execute(
    (k: string, m: boolean, s: boolean) => {
      document.dispatchEvent(
        new KeyboardEvent("keydown", {
          key: k,
          code: k,
          metaKey: m,
          ctrlKey: false,
          shiftKey: s,
          bubbles: true,
        }),
      );
    },
    key,
    meta,
    shift,
  );
  await browser.pause(300);
}

// ---------------------------------------------------------------------------
// Detail Panel
// ---------------------------------------------------------------------------

/** Check if the detail panel is visible */
export async function isDetailPanelVisible(): Promise<boolean> {
  return browser.execute(() => {
    return document.querySelector(".detail-panel") !== null;
  });
}

/** Get the detail panel title text */
export async function getDetailTitle(): Promise<string> {
  return browser.execute(() => {
    const input = document.querySelector(".detail-panel .detail-title") as HTMLInputElement;
    return input?.value ?? input?.textContent ?? "";
  });
}

/** Set the detail panel title */
export async function setDetailTitle(value: string) {
  await browser.execute((v: string) => {
    const input = document.querySelector(".detail-panel .detail-title") as HTMLInputElement;
    if (input) {
      const nativeSet = Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value")?.set;
      if (nativeSet) {
        nativeSet.call(input, v);
        input.dispatchEvent(new Event("input", { bubbles: true }));
      }
    }
  }, value);
  await browser.pause(400); // wait for debounce
}

/** Set the detail panel notes */
export async function setDetailNotes(value: string) {
  await browser.execute((v: string) => {
    const textarea = document.querySelector(".detail-panel textarea") as HTMLTextAreaElement;
    if (textarea) {
      const nativeSet = Object.getOwnPropertyDescriptor(HTMLTextAreaElement.prototype, "value")?.set;
      if (nativeSet) {
        nativeSet.call(textarea, v);
        textarea.dispatchEvent(new Event("input", { bubbles: true }));
      }
    }
  }, value);
  await browser.pause(400); // wait for debounce
}

/** Get the detail panel notes */
export async function getDetailNotes(): Promise<string> {
  return browser.execute(() => {
    const textarea = document.querySelector(".detail-panel textarea") as HTMLTextAreaElement;
    return textarea?.value ?? "";
  });
}

/** Get a detail field value by label */
export async function getDetailFieldValue(label: string): Promise<string> {
  return browser.execute((lbl: string) => {
    const fields = document.querySelectorAll(".detail-field");
    for (const field of fields) {
      const labelEl = field.querySelector(".detail-field-label");
      if (labelEl?.textContent?.includes(lbl)) {
        const valueEl = field.querySelector(".detail-field-value");
        return valueEl?.textContent?.trim() ?? "";
      }
    }
    return "";
  }, label);
}

/** Click a detail field value by label (to open its picker) */
export async function clickDetailField(label: string) {
  await browser.execute((lbl: string) => {
    const fields = document.querySelectorAll(".detail-field");
    for (const field of fields) {
      const labelEl = field.querySelector(".detail-field-label");
      if (labelEl?.textContent?.includes(lbl)) {
        const valueEl = field.querySelector(".detail-field-value span");
        if (valueEl) (valueEl as HTMLElement).click();
        return;
      }
    }
  }, label);
  await browser.pause(200);
}

// ---------------------------------------------------------------------------
// When Picker (Schedule)
// ---------------------------------------------------------------------------

/** Check if the WhenPicker popover is visible */
export async function isWhenPickerVisible(): Promise<boolean> {
  return browser.execute(() => document.querySelector(".when-popover") !== null);
}

/** Click a WhenPicker option by text (Today, This Evening, Someday) */
export async function clickWhenOption(optionText: string) {
  await browser.execute((text: string) => {
    const options = document.querySelectorAll(".when-option");
    for (const opt of options) {
      if (opt.textContent?.includes(text)) {
        (opt as HTMLElement).click();
        return;
      }
    }
    throw new Error(`When option "${text}" not found`);
  }, optionText);
  await browser.pause(300);
}

/** Click the WhenPicker "Clear" button */
export async function clickWhenClear() {
  await browser.execute(() => {
    const clear = document.querySelector(".when-clear");
    if (clear) (clear as HTMLElement).click();
  });
  await browser.pause(300);
}

// ---------------------------------------------------------------------------
// Tag Picker
// ---------------------------------------------------------------------------

/** Click the "+ Add" link in the Tags detail field to open TagPicker */
export async function openTagPicker() {
  await browser.execute(() => {
    // Find all spans in the detail panel and click the one that says "+ Add"
    const spans = document.querySelectorAll(".detail-panel span");
    for (const span of spans) {
      if (span.textContent?.trim() === "+ Add") {
        (span as HTMLElement).click();
        return;
      }
    }
    // Broader search
    const allSpans = document.querySelectorAll("span");
    for (const span of allSpans) {
      if (span.textContent?.trim() === "+ Add") {
        (span as HTMLElement).click();
        return;
      }
    }
  });
  await browser.pause(300);
}

/** Type a new tag name in the TagPicker and press Enter to create it */
export async function createTagInPicker(tagName: string) {
  // Step 1: Find and set the input value
  const found = await browser.execute((name: string) => {
    // Search all text inputs for one with a tag-related placeholder
    const inputs = document.querySelectorAll("input[type='text']") as NodeListOf<HTMLInputElement>;
    for (const input of inputs) {
      if (input.placeholder.toLowerCase().includes("tag") || input.placeholder.includes("tag")) {
        input.focus();
        const nativeSet = Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value")?.set;
        if (nativeSet) {
          nativeSet.call(input, name);
          input.dispatchEvent(new Event("input", { bubbles: true }));
        }
        return true;
      }
    }
    return false;
  }, tagName);

  if (!found) {
    throw new Error("Tag picker input not found - is the tag picker open?");
  }

  await browser.pause(200);

  // Step 2: Press Enter
  await browser.execute(() => {
    const inputs = document.querySelectorAll("input[type='text']") as NodeListOf<HTMLInputElement>;
    for (const input of inputs) {
      if (input.placeholder.toLowerCase().includes("tag") || input.placeholder.includes("tag")) {
        input.dispatchEvent(
          new KeyboardEvent("keydown", { key: "Enter", code: "Enter", bubbles: true }),
        );
        return;
      }
    }
  });
  await browser.pause(500);
}

/** Toggle a tag by name in the open TagPicker */
export async function toggleTagInPicker(tagName: string) {
  await browser.execute((name: string) => {
    // TagPicker renders each tag as a div with a checkbox and span
    const checkboxes = document.querySelectorAll("input[type='checkbox']");
    for (const cb of checkboxes) {
      const row = cb.parentElement;
      if (row?.textContent?.includes(name)) {
        // Dispatch mousedown then click to trigger React's onClick
        row.dispatchEvent(new MouseEvent("mousedown", { bubbles: true }));
        row.dispatchEvent(new MouseEvent("click", { bubbles: true }));
        return;
      }
    }
  }, tagName);
  await browser.pause(300);
}

/** Get tag pill labels visible in the detail panel header */
export async function getDetailTagPills(): Promise<string[]> {
  return browser.execute(() => {
    // TagPill component uses class "tag tag-{variant}", not "tag-pill"
    const pills = document.querySelectorAll(".detail-header .tag");
    return Array.from(pills).map((el) => el.textContent?.trim().replace(/\u00d7$/, "").trim() ?? "");
  });
}

/** Get tag pill labels visible on a task row */
export async function getTaskTagPills(title: string): Promise<string[]> {
  return browser.execute((t: string) => {
    const items = document.querySelectorAll(".task-item");
    for (const item of items) {
      const titleEl = item.querySelector(".task-title");
      if (titleEl?.textContent === t) {
        // TagPill uses class "tag tag-{variant}"
        const pills = item.querySelectorAll(".tag");
        return Array.from(pills).map((el) => el.textContent?.trim().replace(/\u00d7$/, "").trim() ?? "");
      }
    }
    return [];
  }, title);
}

// ---------------------------------------------------------------------------
// Project Picker
// ---------------------------------------------------------------------------

/** Click the Project field value in detail panel to open ProjectPicker */
export async function openProjectPicker() {
  await clickDetailField("Project");
}

/** Select a project by name in the open ProjectPicker */
export async function selectProjectInPicker(projectName: string) {
  await browser.execute((name: string) => {
    // The ProjectPicker renders rows as divs with cursor:pointer style
    // Search more broadly for divs containing the project name
    const allDivs = document.querySelectorAll("div");
    for (const div of allDivs) {
      const style = (div as HTMLElement).style;
      if (style.cursor === "pointer" && div.textContent?.includes(name)) {
        (div as HTMLElement).click();
        return;
      }
    }
    throw new Error(`Project "${name}" not found in picker`);
  }, projectName);
  await browser.pause(300);
}

// ---------------------------------------------------------------------------
// Checklist
// ---------------------------------------------------------------------------

/** Add a checklist item in the detail panel */
export async function addChecklistItem(title: string) {
  await browser.execute((t: string) => {
    const input = document.querySelector("input[placeholder='New item']") as HTMLInputElement;
    if (input) {
      const nativeSet = Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value")?.set;
      if (nativeSet) {
        nativeSet.call(input, t);
        input.dispatchEvent(new Event("input", { bubbles: true }));
      }
      input.dispatchEvent(
        new KeyboardEvent("keydown", { key: "Enter", code: "Enter", bubbles: true }),
      );
    }
  }, title);
  await browser.pause(300);
}

/** Get all checklist item texts */
export async function getChecklistItems(): Promise<string[]> {
  return browser.execute(() => {
    const items = document.querySelectorAll(".cl-item");
    return Array.from(items).map((el) => {
      // Get span text (not the checkbox or delete btn)
      const spans = el.querySelectorAll("span");
      for (const span of spans) {
        if (!span.classList.contains("cl-delete-btn") && span.textContent?.trim()) {
          return span.textContent.trim();
        }
      }
      return "";
    }).filter(Boolean);
  });
}

/** Toggle a checklist item by its text */
export async function toggleChecklistItem(title: string) {
  await browser.execute((t: string) => {
    const items = document.querySelectorAll(".cl-item");
    for (const item of items) {
      if (item.textContent?.includes(t)) {
        const check = item.querySelector(".cl-check");
        if (check) (check as HTMLElement).click();
        return;
      }
    }
  }, title);
  await browser.pause(300);
}

/** Check if a checklist item is done */
export async function isChecklistItemDone(title: string): Promise<boolean> {
  return browser.execute((t: string) => {
    const items = document.querySelectorAll(".cl-item");
    for (const item of items) {
      if (item.textContent?.includes(t)) {
        return item.querySelector(".cl-check.done") !== null;
      }
    }
    return false;
  }, title);
}

/** Delete a checklist item by clicking its X button */
export async function deleteChecklistItem(title: string) {
  await browser.execute((t: string) => {
    const items = document.querySelectorAll(".cl-item");
    for (const item of items) {
      if (item.textContent?.includes(t)) {
        const btn = item.querySelector(".cl-delete-btn");
        if (btn) (btn as HTMLElement).click();
        return;
      }
    }
  }, title);
  await browser.pause(300);
}

// ---------------------------------------------------------------------------
// Context Menu
// ---------------------------------------------------------------------------

/** Right-click a task to open context menu */
export async function rightClickTask(title: string) {
  await browser.execute((t: string) => {
    const items = document.querySelectorAll(".task-item");
    for (const item of items) {
      const titleEl = item.querySelector(".task-title");
      if (titleEl?.textContent === t) {
        (item as HTMLElement).dispatchEvent(
          new MouseEvent("contextmenu", {
            bubbles: true,
            clientX: 300,
            clientY: 200,
          }),
        );
        return;
      }
    }
    throw new Error(`Task "${t}" not found for right-click`);
  }, title);
  await browser.pause(200);
}

/** Click a context menu item by label */
export async function clickContextMenuItem(label: string) {
  await browser.execute((lbl: string) => {
    // Context menu renders items as divs with cursor:pointer and child spans
    // Search all elements for one whose text content matches the label
    const allDivs = document.querySelectorAll("div");
    for (const div of allDivs) {
      const style = (div as HTMLElement).style;
      if (style.cursor === "pointer") {
        // Check if any child span has the exact label text
        const spans = div.querySelectorAll("span");
        for (const span of spans) {
          if (span.textContent?.trim() === lbl) {
            (div as HTMLElement).click();
            return;
          }
        }
      }
    }
    throw new Error(`Context menu item "${lbl}" not found`);
  }, label);
  await browser.pause(300);
}

/** Check if context menu is visible (has fixed-positioned element with min-width 200) */
export async function isContextMenuVisible(): Promise<boolean> {
  return browser.execute(() => {
    // Context menu uses inline styles with position: fixed and minWidth: 200
    const els = document.querySelectorAll("[style*='position: fixed']");
    for (const el of els) {
      if ((el as HTMLElement).style.minWidth === "200px" || el.innerHTML.includes("flex: 1")) {
        return true;
      }
    }
    return false;
  });
}

// ---------------------------------------------------------------------------
// Command Palette
// ---------------------------------------------------------------------------

/** Check if command palette is open */
export async function isCommandPaletteOpen(): Promise<boolean> {
  return browser.execute(() => {
    const palette = document.querySelector(".cmd-palette");
    return palette?.classList.contains("open") ?? false;
  });
}

/** Type in the command palette search input */
export async function typeInCommandPalette(text: string) {
  await browser.execute((t: string) => {
    const input = document.querySelector(".cmd-input") as HTMLInputElement;
    if (input) {
      const nativeSet = Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value")?.set;
      if (nativeSet) {
        nativeSet.call(input, t);
        input.dispatchEvent(new Event("input", { bubbles: true }));
      }
    }
  }, text);
  await browser.pause(200);
}

/** Click a command palette item by label */
export async function clickCommandPaletteItem(label: string) {
  await browser.execute((lbl: string) => {
    const items = document.querySelectorAll(".cmd-item");
    for (const item of items) {
      const labelEl = item.querySelector(".cmd-item-label");
      if (labelEl?.textContent?.includes(lbl)) {
        (item as HTMLElement).click();
        return;
      }
    }
    throw new Error(`Command palette item "${lbl}" not found`);
  }, label);
  await browser.pause(300);
}

/** Get command palette item labels */
export async function getCommandPaletteItems(): Promise<string[]> {
  return browser.execute(() => {
    const items = document.querySelectorAll(".cmd-item-label");
    return Array.from(items).map((el) => el.textContent?.trim() ?? "");
  });
}

// ---------------------------------------------------------------------------
// Bulk Action Bar
// ---------------------------------------------------------------------------

/** Check if the bulk action bar is visible */
export async function isBulkBarVisible(): Promise<boolean> {
  return browser.execute(() => {
    const bar = document.querySelector(".bulk-bar-btn");
    return bar !== null;
  });
}

/** Get the bulk bar selected count text */
export async function getBulkBarCount(): Promise<string> {
  return browser.execute(() => {
    const bars = document.querySelectorAll("[style*='position: fixed'][style*='bottom']");
    for (const bar of bars) {
      const countEl = bar.querySelector("span[style*='font-weight: 700']");
      if (countEl) return countEl.textContent?.trim() ?? "";
    }
    return "";
  });
}

/** Click a bulk action bar button by its text content */
export async function clickBulkAction(text: string) {
  await browser.execute((t: string) => {
    const buttons = document.querySelectorAll(".bulk-bar-btn");
    for (const btn of buttons) {
      if (btn.textContent?.includes(t)) {
        (btn as HTMLElement).click();
        return;
      }
    }
    throw new Error(`Bulk action "${t}" not found`);
  }, text);
  await browser.pause(500);
}

// ---------------------------------------------------------------------------
// Logbook
// ---------------------------------------------------------------------------

/** Get logbook task titles */
export async function getLogbookTitles(): Promise<string[]> {
  return browser.execute(() => {
    const rows = document.querySelectorAll(".logbook-row .task-title");
    return Array.from(rows).map((el) => el.textContent?.trim() ?? "").filter(Boolean);
  });
}

/** Check if a logbook entry has "Cancelled" pill */
export async function isLogbookEntryCancelled(title: string): Promise<boolean> {
  return browser.execute((t: string) => {
    const rows = document.querySelectorAll(".logbook-row");
    for (const row of rows) {
      const titleEl = row.querySelector(".task-title");
      if (titleEl?.textContent?.trim() === t) {
        return row.textContent?.includes("Cancelled") ?? false;
      }
    }
    return false;
  }, title);
}

/** Click "Reopen" button on a logbook entry */
export async function clickReopenInLogbook(title: string) {
  await browser.execute((t: string) => {
    const rows = document.querySelectorAll(".logbook-row");
    for (const row of rows) {
      const titleEl = row.querySelector(".task-title");
      if (titleEl?.textContent?.trim() === t) {
        const btn = row.querySelector(".reopen-btn");
        if (btn) (btn as HTMLElement).click();
        return;
      }
    }
  }, title);
  await browser.pause(300);
}

// ---------------------------------------------------------------------------
// Project & Section
// ---------------------------------------------------------------------------

/** Create a project via the command palette */
export async function createProjectViaPalette(title?: string) {
  await pressKeys("O", true, true); // Open command palette
  await browser.pause(300);
  await clickCommandPaletteItem("New Project");
  await browser.pause(500);

  // If a title is specified, we need to navigate to the project and rename it
  // The command palette creates "New Project" by default
  if (title && title !== "New Project") {
    // The project was just created - find it in sidebar and navigate to it
    // Then we'd need to rename via the Toolbar or inline
    // For now, projects are created with "New Project" title
  }
}

/** Get section headers in the current project view */
export async function getSectionHeaders(): Promise<string[]> {
  return browser.execute(() => {
    const headers = document.querySelectorAll(".section-header-title");
    return Array.from(headers).map((el) => el.textContent?.trim() ?? "").filter(Boolean);
  });
}

/** Click "Add Section" button */
export async function clickAddSection() {
  await browser.execute(() => {
    const buttons = document.querySelectorAll("button");
    for (const btn of buttons) {
      if (btn.textContent?.includes("Add Section")) {
        btn.click();
        return;
      }
    }
  });
  await browser.pause(300);
}

/** Click a section header to toggle collapse */
export async function clickSectionHeader(title: string) {
  await browser.execute((t: string) => {
    const headers = document.querySelectorAll(".section-header");
    for (const header of headers) {
      const titleEl = header.querySelector(".section-header-title");
      if (titleEl?.textContent?.trim() === t) {
        (header as HTMLElement).click();
        return;
      }
    }
  }, title);
  await browser.pause(200);
}

/** Check if a section's chevron is in collapsed state */
export async function isSectionCollapsed(title: string): Promise<boolean> {
  return browser.execute((t: string) => {
    const headers = document.querySelectorAll(".section-header");
    for (const header of headers) {
      const titleEl = header.querySelector(".section-header-title");
      if (titleEl?.textContent?.trim() === t) {
        const chevron = header.querySelector(".section-header-chevron");
        return chevron?.classList.contains("collapsed") ?? false;
      }
    }
    return false;
  }, title);
}

/** Get section header task count */
export async function getSectionCount(title: string): Promise<number> {
  return browser.execute((t: string) => {
    const headers = document.querySelectorAll(".section-header");
    for (const header of headers) {
      const titleEl = header.querySelector(".section-header-title");
      if (titleEl?.textContent?.trim() === t) {
        const count = header.querySelector(".section-header-count");
        return parseInt(count?.textContent ?? "0", 10);
      }
    }
    return 0;
  }, title);
}

/** Check if progress bar exists */
export async function isProgressBarVisible(): Promise<boolean> {
  return browser.execute(() => {
    return document.querySelector(".progress-bar") !== null ||
           document.querySelector("[class*='progress']") !== null;
  });
}

// ---------------------------------------------------------------------------
// Sidebar helpers
// ---------------------------------------------------------------------------

/** Get sidebar badge count */
export async function getSidebarBadge(viewName: string): Promise<string> {
  return browser.execute((name: string) => {
    const items = document.querySelectorAll(".sidebar-item");
    for (const item of items) {
      if (item.textContent?.includes(name)) {
        const badge = item.querySelector(".sidebar-badge");
        return badge?.textContent ?? "0";
      }
    }
    return "0";
  }, viewName);
}

// ---------------------------------------------------------------------------
// Inline Editor
// ---------------------------------------------------------------------------

/** Check if inline editor is open */
export async function isInlineEditorOpen(): Promise<boolean> {
  return browser.execute(() => {
    return document.querySelector(".task-item.editing") !== null;
  });
}

/** Get inline editor title input value */
export async function getInlineEditorTitle(): Promise<string> {
  return browser.execute(() => {
    const input = document.querySelector(".task-title-input") as HTMLInputElement;
    return input?.value ?? "";
  });
}

/** Set inline editor title */
export async function setInlineEditorTitle(value: string) {
  await browser.execute((v: string) => {
    const input = document.querySelector(".task-title-input") as HTMLInputElement;
    if (input) {
      const nativeSet = Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value")?.set;
      if (nativeSet) {
        nativeSet.call(input, v);
        input.dispatchEvent(new Event("input", { bubbles: true }));
      }
    }
  }, value);
  await browser.pause(200);
}

/** Get inline editor notes textarea value */
export async function getInlineEditorNotes(): Promise<string> {
  return browser.execute(() => {
    const textarea = document.querySelector(".task-item.editing .task-inline-notes-input") as HTMLTextAreaElement;
    return textarea?.value ?? "";
  });
}

/** Set inline editor notes textarea value */
export async function setInlineEditorNotes(value: string) {
  await browser.execute((v: string) => {
    const textarea = document.querySelector(".task-item.editing .task-inline-notes-input") as HTMLTextAreaElement;
    if (textarea) {
      const nativeSet = Object.getOwnPropertyDescriptor(HTMLTextAreaElement.prototype, "value")?.set;
      if (nativeSet) {
        nativeSet.call(textarea, v);
        textarea.dispatchEvent(new Event("input", { bubbles: true }));
      }
    }
  }, value);
  await browser.pause(200);
}

/** Press Escape on the inline editor notes textarea */
export async function escapeInlineEditorNotes() {
  await browser.execute(() => {
    const textarea = document.querySelector(".task-item.editing .task-inline-notes-input") as HTMLTextAreaElement;
    if (textarea) {
      textarea.dispatchEvent(
        new KeyboardEvent("keydown", { key: "Escape", code: "Escape", bubbles: true }),
      );
    }
  });
  await browser.pause(300);
}

// ---------------------------------------------------------------------------
// Triage Actions (Inbox view)
// ---------------------------------------------------------------------------

/** Click a triage action button on a task (Today star, Schedule, Someday, Move) */
export async function clickTriageAction(title: string, action: "today" | "schedule" | "someday" | "move") {
  await browser.execute((t: string, a: string) => {
    const items = document.querySelectorAll(".task-item");
    for (const item of items) {
      const titleEl = item.querySelector(".task-title");
      if (titleEl?.textContent === t) {
        const buttons = item.querySelectorAll(".task-action-btn");
        const titleMap: Record<string, string> = {
          today: "Schedule for Today",
          schedule: "Schedule",
          someday: "Someday",
          move: "Move to Project",
        };
        for (const btn of buttons) {
          if ((btn as HTMLElement).title === titleMap[a]) {
            (btn as HTMLElement).click();
            return;
          }
        }
      }
    }
  }, title, action);
  await browser.pause(300);
}

// ---------------------------------------------------------------------------
// Generic helpers
// ---------------------------------------------------------------------------

/** Count elements matching a selector */
export async function countElements(selector: string): Promise<number> {
  return browser.execute((sel: string) => {
    return document.querySelectorAll(sel).length;
  }, selector);
}

/** Check if an element exists */
export async function elementExists(selector: string): Promise<boolean> {
  return browser.execute((sel: string) => {
    return document.querySelector(sel) !== null;
  }, selector);
}

/** Check if any element contains specific text */
export async function elementWithTextExists(selector: string, text: string): Promise<boolean> {
  return browser.execute((sel: string, txt: string) => {
    const els = document.querySelectorAll(sel);
    for (const el of els) {
      if (el.textContent?.includes(txt)) return true;
    }
    return false;
  }, selector, text);
}

/** Wait for an element to appear */
export async function waitForElement(selector: string, timeout = 5000) {
  await browser.waitUntil(
    async () => browser.execute((sel: string) => document.querySelector(sel) !== null, selector),
    { timeout, timeoutMsg: `Element "${selector}" did not appear within ${timeout}ms` },
  );
}

/** Wait for an element to disappear */
export async function waitForElementGone(selector: string, timeout = 5000) {
  await browser.waitUntil(
    async () => browser.execute((sel: string) => document.querySelector(sel) === null, selector),
    { timeout, timeoutMsg: `Element "${selector}" did not disappear within ${timeout}ms` },
  );
}
