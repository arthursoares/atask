import {
  waitForAppReady,
  navigateTo,
  createTaskViaUI,
  clickTask,
  isDetailPanelVisible,
  getDetailTitle,
  setDetailTitle,
  setDetailNotes,
  getDetailNotes,
  getDetailFieldValue,
  clickDetailField,
  openProjectPicker,
  selectProjectInPicker,
  pressKeys,
  getTaskTitles,
  createProjectViaPalette,
} from "./helpers";

describe("Detail Panel Interactions", () => {
  before(async () => {
    await waitForAppReady();
    await navigateTo("Inbox");
  });

  describe("Title editing", () => {
    it("should create a task and open detail panel", async () => {
      await createTaskViaUI("Detail Edit Task");
      await clickTask("Detail Edit Task");
      expect(await isDetailPanelVisible()).toBe(true);
    });

    it("should show the task title in the detail panel input", async () => {
      const title = await getDetailTitle();
      expect(title).toBe("Detail Edit Task");
    });

    it("should edit the task title via detail panel", async () => {
      await setDetailTitle("Detail Edit Task Renamed");
      await browser.pause(500); // wait for debounce to save

      // Verify the title persisted
      const title = await getDetailTitle();
      expect(title).toBe("Detail Edit Task Renamed");
    });

    it("should reflect the renamed title in the task list", async () => {
      // Close and check
      await pressKeys("Escape");
      await browser.pause(300);
      const titles = await getTaskTitles();
      expect(titles).toContain("Detail Edit Task Renamed");
    });
  });

  describe("Notes editing", () => {
    it("should open detail panel and add notes", async () => {
      await clickTask("Detail Edit Task Renamed");
      expect(await isDetailPanelVisible()).toBe(true);

      await setDetailNotes("These are some test notes for the task.");
      await browser.pause(500); // debounce

      const notes = await getDetailNotes();
      expect(notes).toBe("These are some test notes for the task.");
    });

    it("should persist notes after closing and reopening", async () => {
      await pressKeys("Escape");
      await browser.pause(200);
      await clickTask("Detail Edit Task Renamed");
      await browser.pause(300);

      const notes = await getDetailNotes();
      expect(notes).toBe("These are some test notes for the task.");
    });
  });

  describe("Detail panel field display", () => {
    it("should show Schedule field", async () => {
      const schedule = await getDetailFieldValue("Schedule");
      expect(schedule).toBeDefined();
      expect(schedule.length).toBeGreaterThan(0);
    });

    it("should show Start Date field", async () => {
      const startDate = await getDetailFieldValue("Start Date");
      expect(startDate).toBeDefined();
    });

    it("should show Deadline field", async () => {
      const deadline = await getDetailFieldValue("Deadline");
      expect(deadline).toBeDefined();
    });

    it("should show Tags field with + Add link", async () => {
      const tags = await getDetailFieldValue("Tags");
      expect(tags).toContain("Add");
    });

    it("should show Notes field", async () => {
      // Notes field has a textarea
      const hasTextarea = await browser.execute(() => {
        return document.querySelector(".detail-panel textarea") !== null;
      });
      expect(hasTextarea).toBe(true);
    });

    it("should show Checklist field with new item input", async () => {
      const hasInput = await browser.execute(() => {
        return document.querySelector("input[placeholder='New item']") !== null;
      });
      expect(hasInput).toBe(true);
    });
  });

  describe("Project assignment via Detail Panel", () => {
    it("should ensure a project exists", async () => {
      await pressKeys("Escape"); // close detail
      await browser.pause(200);

      // Create a project via command palette if none exists
      await createProjectViaPalette();
      await browser.pause(500);

      // Navigate back to Inbox
      await navigateTo("Inbox");
      await browser.pause(200);
    });

    it("should assign task to project via ProjectPicker", async () => {
      await clickTask("Detail Edit Task Renamed");
      expect(await isDetailPanelVisible()).toBe(true);

      await openProjectPicker();
      await browser.pause(200);

      // Select "New Project" from the picker
      await selectProjectInPicker("New Project");
      await browser.pause(300);

      // Verify the project field shows the project name
      const projectValue = await getDetailFieldValue("Project");
      expect(projectValue).toContain("New Project");
    });

    // TODO: click-outside handler on ProjectPicker popover intercepts synthetic mousedown before row onClick
    it.skip("should remove project assignment (set to Inbox/No Project)", async () => {
      // Click the project field to open the picker
      await clickDetailField("Project");
      await browser.pause(300);

      // Click the "Inbox (No Project)" option - use mousedown+mouseup+click sequence
      const clicked = await browser.execute(() => {
        const allSpans = document.querySelectorAll("span");
        for (const span of allSpans) {
          if (span.textContent?.includes("No Project")) {
            const row = span.parentElement;
            if (row) {
              // Full click sequence so React picks it up
              row.dispatchEvent(new MouseEvent("mousedown", { bubbles: true }));
              row.dispatchEvent(new MouseEvent("mouseup", { bubbles: true }));
              row.dispatchEvent(new MouseEvent("click", { bubbles: true }));
              return true;
            }
          }
        }
        return false;
      });
      expect(clicked).toBe(true);
      await browser.pause(500);

      const projectValue = await getDetailFieldValue("Project");
      expect(projectValue).toContain("None");
    });
  });

  describe("Escape to close", () => {
    it("should close the detail panel with Escape", async () => {
      expect(await isDetailPanelVisible()).toBe(true);
      await pressKeys("Escape");
      await browser.pause(200);
      expect(await isDetailPanelVisible()).toBe(false);
    });
  });

  after(async () => {
    await navigateTo("Inbox");
  });
});
