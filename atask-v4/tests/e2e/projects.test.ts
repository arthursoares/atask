import {
  waitForAppReady,
  resetDatabase,
  navigateTo,
  createTaskViaUI,
  getTaskTitles,
  clickTask,
  pressKeys,
  getSidebarLabels,
  clickAddSection,
  getSectionHeaders,
  clickSectionHeader,
  isSectionCollapsed,
  getSectionCount,
  getTaskCount,
  createProjectViaPalette,
  isDetailPanelVisible,
  getSidebarBadge,
} from "./helpers";

describe("Projects & Sections", () => {
  before(async () => {
    await waitForAppReady();
  });

  beforeEach(async () => {
    await resetDatabase();
    await waitForAppReady();
  });

  describe("Project creation", () => {
    it("should create a project via command palette", async () => {
      await createProjectViaPalette();
      await browser.pause(500);

      // Project should appear in the sidebar
      const labels = await getSidebarLabels();
      expect(labels.some((l) => l.includes("New Project"))).toBe(true);
    });

    it("should navigate to the new project via sidebar", async () => {
      await navigateTo("New Project");
      await browser.pause(300);
      // We successfully navigated to the project view
      // (task count depends on prior test state, so just verify navigation worked)
      const title = await browser.execute(() => {
        const el = document.querySelector(".app-view-title");
        return el?.textContent ?? "";
      });
      expect(title).toContain("New Project");
    });
  });

  describe("Tasks in projects", () => {
    it("should create tasks inside a project", async () => {
      // We should already be on the project view from previous test
      await createTaskViaUI("Project Task Alpha");
      await createTaskViaUI("Project Task Beta");
      const titles = await getTaskTitles();
      expect(titles).toContain("Project Task Alpha");
      expect(titles).toContain("Project Task Beta");
    });

    it("should select a project task and show detail panel", async () => {
      await clickTask("Project Task Alpha");
      expect(await isDetailPanelVisible()).toBe(true);
      // Close it
      await pressKeys("Escape");
      await browser.pause(200);
    });
  });

  describe("Sections", () => {
    it("should add a section to the project", async () => {
      await clickAddSection();
      await browser.pause(300);
      const headers = await getSectionHeaders();
      expect(headers).toContain("New Section");
    });

    it("should add another section", async () => {
      await clickAddSection();
      await browser.pause(300);
      const headers = await getSectionHeaders();
      // Should have 2 "New Section" headers (both default name)
      expect(headers.length).toBeGreaterThanOrEqual(2);
    });

    it("should collapse a section", async () => {
      await clickSectionHeader("New Section");
      const collapsed = await isSectionCollapsed("New Section");
      expect(collapsed).toBe(true);
    });

    it("should expand a collapsed section", async () => {
      await clickSectionHeader("New Section");
      const collapsed = await isSectionCollapsed("New Section");
      expect(collapsed).toBe(false);
    });
  });

  describe("Sidebar project badge", () => {
    it("should show task count badge next to project in sidebar", async () => {
      // We created 2 tasks in the project (plus any from previous suites)
      const badge = await getSidebarBadge("New Project");
      // Badge should be at least 2 (from our 2 created tasks)
      expect(parseInt(badge, 10)).toBeGreaterThanOrEqual(2);
    });
  });

  after(async () => {
    // Navigate back to inbox for other tests
    await navigateTo("Inbox");
  });
});
