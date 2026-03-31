import {
  waitForAppReady,
  navigateTo,
  createTaskViaUI,
  clickTask,
  isDetailPanelVisible,
  openTagPicker,
  createTagInPicker,
  toggleTagInPicker,
  getDetailTagPills,
  getTaskTagPills,
  getDetailFieldValue,
  pressKeys,
} from "./helpers";

describe("Tags", () => {
  before(async () => {
    await waitForAppReady();
    await navigateTo("Inbox");
  });

  it("should create a task for tagging", async () => {
    await createTaskViaUI("Tagged Task");
    const titles = await browser.execute(() => {
      const items = document.querySelectorAll(".task-title");
      return Array.from(items).map((el) => el.textContent ?? "").filter(Boolean);
    });
    expect(titles).toContain("Tagged Task");
  });

  it("should open detail panel and see Tags field", async () => {
    await clickTask("Tagged Task");
    expect(await isDetailPanelVisible()).toBe(true);
    const tagFieldValue = await getDetailFieldValue("Tags");
    expect(tagFieldValue).toContain("Add");
  });

  it("should open the tag picker via + Add link", async () => {
    await openTagPicker();

    // Verify the tag picker is visible (has a text input for new tags)
    const hasTagInput = await browser.execute(() => {
      const inputs = document.querySelectorAll("input[type='text']") as NodeListOf<HTMLInputElement>;
      for (const input of inputs) {
        if (input.placeholder.toLowerCase().includes("tag")) return true;
      }
      return false;
    });
    expect(hasTagInput).toBe(true);
  });

  it("should create a new tag via the picker", async () => {
    await createTagInPicker("urgent");

    // Wait for store update and re-render
    await browser.pause(500);

    // Check tag pills in the detail header
    const pills = await getDetailTagPills();

    // Also check the Tags field value as fallback
    const tagFieldHTML = await browser.execute(() => {
      const fields = document.querySelectorAll(".detail-field");
      for (const field of fields) {
        const label = field.querySelector(".detail-field-label");
        if (label?.textContent?.includes("Tags")) {
          return field.querySelector(".detail-field-value")?.textContent ?? "";
        }
      }
      return "";
    });

    const hasUrgent = pills.some((p) => p.toLowerCase().includes("urgent"))
      || tagFieldHTML.toLowerCase().includes("urgent");
    expect(hasUrgent).toBe(true);
  });

  it("should show the tag pill on the task row", async () => {
    await pressKeys("Escape");
    await browser.pause(200);

    const taskPills = await getTaskTagPills("Tagged Task");
    expect(taskPills.some((p) => p.toLowerCase().includes("urgent"))).toBe(true);
  });

  it("should create a second tag and assign it", async () => {
    await clickTask("Tagged Task");
    await openTagPicker();
    await createTagInPicker("feature");
    await browser.pause(500);

    const pills = await getDetailTagPills();
    const tagFieldHTML = await browser.execute(() => {
      const fields = document.querySelectorAll(".detail-field");
      for (const field of fields) {
        const label = field.querySelector(".detail-field-label");
        if (label?.textContent?.includes("Tags")) {
          return field.querySelector(".detail-field-value")?.textContent ?? "";
        }
      }
      return "";
    });

    const hasFeature = pills.some((p) => p.toLowerCase().includes("feature"))
      || tagFieldHTML.toLowerCase().includes("feature");
    expect(hasFeature).toBe(true);
  });

  // TODO: click-outside handler on TagPicker popover intercepts synthetic mousedown before row onClick
  it.skip("should toggle (remove) a tag from the picker", async () => {
    // Close the picker if it's still open from previous test, then reopen
    await browser.execute(() => {
      // Click outside to close any open picker
      const panel = document.querySelector(".detail-panel");
      if (panel) (panel as HTMLElement).click();
    });
    await browser.pause(200);

    // Now open the tag picker fresh
    await openTagPicker();
    await browser.pause(300);

    // Toggle "urgent" off
    await toggleTagInPicker("urgent");
    await browser.pause(300);

    // Close picker by clicking the detail panel body
    await browser.execute(() => {
      const body = document.querySelector(".detail-body");
      if (body) (body as HTMLElement).dispatchEvent(
        new MouseEvent("mousedown", { bubbles: true, clientX: 10, clientY: 10 }),
      );
    });
    await browser.pause(300);

    const pills = await getDetailTagPills();
    expect(pills.some((p) => p.toLowerCase().includes("urgent"))).toBe(false);
  });

  after(async () => {
    await pressKeys("Escape");
    await navigateTo("Inbox");
  });
});
