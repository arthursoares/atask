import { waitForAppReady, resetDatabase, navigateTo, getViewTitle, pressKeys, elementExists } from "./helpers";

describe("Navigation", () => {
  before(async () => {
    await waitForAppReady();
  });

  beforeEach(async () => {
    await resetDatabase();
    await waitForAppReady();
  });

  it("should start on Inbox view", async () => {
    const title = await getViewTitle();
    expect(title).toContain("Inbox");
  });

  it("should navigate to Today via sidebar", async () => {
    await navigateTo("Today");
    await browser.pause(500);
    const title = await getViewTitle();
    expect(title).toContain("Today");
  });

  it("should still have sidebar items after navigation", async () => {
    // Debug: check if the sidebar DOM is intact after navigating
    const count = await browser.execute(() => {
      return document.querySelectorAll(".sidebar-item").length;
    });
    const html = await browser.execute(() => {
      const sidebar = document.querySelector(".sidebar");
      return sidebar ? sidebar.innerHTML.substring(0, 500) : "NO SIDEBAR";
    });
    console.log(`Sidebar items: ${count}, HTML: ${html}`);
    expect(count).toBeGreaterThan(0);
  });

  it("should navigate to Upcoming via sidebar", async () => {
    await navigateTo("Upcoming");
    await browser.pause(500);
    const title = await getViewTitle();
    expect(title).toContain("Upcoming");
  });

  it("should navigate to Someday via sidebar", async () => {
    await navigateTo("Someday");
    const title = await getViewTitle();
    expect(title).toContain("Someday");
  });

  it("should navigate to Logbook via sidebar", async () => {
    await navigateTo("Logbook");
    const title = await getViewTitle();
    expect(title).toContain("Logbook");
  });

  it("should navigate via ⌘1-5 keyboard shortcuts", async () => {
    await pressKeys("1", true);
    expect(await getViewTitle()).toContain("Inbox");

    await pressKeys("2", true);
    expect(await getViewTitle()).toContain("Today");

    await pressKeys("3", true);
    expect(await getViewTitle()).toContain("Upcoming");

    await pressKeys("4", true);
    expect(await getViewTitle()).toContain("Someday");

    await pressKeys("5", true);
    expect(await getViewTitle()).toContain("Logbook");
  });

  it("should toggle sidebar with ⌘/", async () => {
    expect(await elementExists(".sidebar")).toBe(true);
    await pressKeys("/", true);
    expect(await elementExists(".sidebar")).toBe(false);
    await pressKeys("/", true);
    expect(await elementExists(".sidebar")).toBe(true);
  });

  after(async () => {
    await navigateTo("Inbox");
  });
});
