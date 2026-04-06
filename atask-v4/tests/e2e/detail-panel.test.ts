import {
  waitForAppReady,
  resetDatabase,
  navigateTo,
  createTaskViaUI,
  clickTask,
  isDetailPanelVisible,
  getDetailTitle,
  elementExists,
  pressKeys,
} from "./helpers";

describe("Detail Panel", () => {
  before(async () => {
    await waitForAppReady();
    await navigateTo("Inbox");
    await createTaskViaUI("Detail Test");
  });

  beforeEach(async () => {
    await resetDatabase();
    await waitForAppReady();
  });

  it("should open when clicking a task", async () => {
    await clickTask("Detail Test");
    expect(await isDetailPanelVisible()).toBe(true);
    expect(await getDetailTitle()).toBe("Detail Test");
  });

  it("should show field labels", async () => {
    const labels = await browser.execute(() => {
      const els = document.querySelectorAll(".detail-field-label");
      return Array.from(els).map((el) => el.textContent ?? "");
    });
    // Labels use CSS text-transform:uppercase, so textContent is lowercase
    const lower = labels.map((l: string) => l.toLowerCase());
    expect(lower.some((l: string) => l.includes("project"))).toBe(true);
    expect(lower.some((l: string) => l.includes("schedule"))).toBe(true);
    expect(lower.some((l: string) => l.includes("checklist"))).toBe(true);
  });

  it("should have a notes textarea", async () => {
    expect(await elementExists(".detail-body textarea")).toBe(true);
  });

  it("should have a checklist section", async () => {
    expect(await elementExists(".detail-field-label")).toBe(true);
  });

  it("should close with Escape", async () => {
    await pressKeys("Escape");
    expect(await isDetailPanelVisible()).toBe(false);
  });
});
