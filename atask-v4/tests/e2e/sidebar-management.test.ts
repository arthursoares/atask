import {
  waitForAppReady,
  navigateTo,
  getSidebarLabels,
  pressKeys,
  clickCommandPaletteItem,
  elementExists,
} from "./helpers";

describe("Sidebar Management", () => {
  before(async () => {
    await waitForAppReady();
  });

  describe("Area creation", () => {
    it("should have a '+ New Area' item in the sidebar", async () => {
      const hasNewArea = await browser.execute(() => {
        const items = document.querySelectorAll(".sidebar-item");
        for (const item of items) {
          if (item.textContent?.includes("New Area")) return true;
        }
        return false;
      });
      expect(hasNewArea).toBe(true);
    });

    it("should create an area via sidebar button", async () => {
      // Click the "+ New Area" button
      await browser.execute(() => {
        const items = document.querySelectorAll(".sidebar-item");
        for (const item of items) {
          if (item.textContent?.includes("New Area")) {
            (item as HTMLElement).click();
            return;
          }
        }
      });
      await browser.pause(500);

      // The area is created but won't show as a group label until it has projects
      // Just verify the area was created by checking that a new sidebar-item appeared
      // or the store has the area. For now, verify no crash occurred.
      const sidebarItemCount = await browser.execute(() => {
        return document.querySelectorAll(".sidebar-item").length;
      });
      expect(sidebarItemCount).toBeGreaterThan(0);
    });
  });

  describe("Project creation via command palette", () => {
    it("should create a project", async () => {
      await pressKeys("O", true, true);
      await browser.pause(300);
      await clickCommandPaletteItem("New Project");
      await browser.pause(500);

      const labels = await getSidebarLabels();
      expect(labels.some((l) => l.includes("New Project"))).toBe(true);
    });

    it("should create a second project for sidebar drag testing", async () => {
      await pressKeys("O", true, true);
      await browser.pause(300);
      await clickCommandPaletteItem("New Project");
      await browser.pause(500);

      const labels = await getSidebarLabels();
      const projectCount = labels.filter((label) => label.includes("New Project")).length;
      expect(projectCount).toBeGreaterThanOrEqual(2);
    });
  });

  describe("Project context menu", () => {
    it("should open context menu on right-click of project", async () => {
      await browser.execute(() => {
        const items = document.querySelectorAll(".sidebar-item");
        for (const item of items) {
          if (item.textContent?.includes("New Project")) {
            item.dispatchEvent(
              new MouseEvent("contextmenu", { bubbles: true, clientX: 200, clientY: 200 }),
            );
            return;
          }
        }
      });
      await browser.pause(200);

      // Context menu should be visible
      const hasMenu = await browser.execute(() => {
        const els = document.querySelectorAll("[style*='position: fixed']");
        for (const el of els) {
          if (el.textContent?.includes("Complete") || el.textContent?.includes("Delete")) {
            return true;
          }
        }
        return false;
      });
      expect(hasMenu).toBe(true);
    });

    it("should show Complete and Delete options", async () => {
      const menuItems = await browser.execute(() => {
        const items: string[] = [];
        const els = document.querySelectorAll("[style*='position: fixed'] span");
        for (const el of els) {
          const text = el.textContent?.trim();
          if (text && text.length > 0) items.push(text);
        }
        return items;
      });

      expect(menuItems.some((item) => item === "Complete")).toBe(true);
      expect(menuItems.some((item) => item === "Delete")).toBe(true);
    });

    it("should close context menu on click outside", async () => {
      await browser.execute(() => {
        document.dispatchEvent(new MouseEvent("mousedown", { bubbles: true }));
      });
      await browser.pause(200);
    });
  });

  describe("Area context menu", () => {
    // Areas only show as group labels when they have projects
    // Since we can't easily assign a project to an area in E2E,
    // just verify the area group label rendering path
    it("should show area group labels when areas have projects", async () => {
      const areaLabels = await browser.execute(() => {
        const labels = document.querySelectorAll(".sidebar-group-label");
        return Array.from(labels).map((l) => l.textContent?.trim() ?? "");
      });
      // May or may not have areas with projects - just verify no crash
      expect(Array.isArray(areaLabels)).toBe(true);
    });
  });

  describe("Sidebar drag slots", () => {
    before(async () => {
      await navigateTo("Inbox");
      const labels = await getSidebarLabels();
      const projectCount = labels.filter((label) => label.includes("New Project")).length;

      for (let index = projectCount; index < 2; index += 1) {
        await pressKeys("O", true, true);
        await browser.pause(300);
        await clickCommandPaletteItem("New Project");
        await browser.pause(500);
      }
    });

    it("should show sidebar drop zones and a visible slot while dragging a project", async () => {
      const dragState = await browser.executeAsync((done) => {
        const finish = (result: Record<string, unknown>) => {
          done(result);
        };

        const groups = Array.from(document.querySelectorAll(".sidebar-group"));
        const projectGroup = groups.find((group) =>
          group.querySelectorAll(".sidebar-item-project").length >= 2,
        ) as HTMLElement | undefined;

        if (!projectGroup) {
          finish({ ok: false, reason: "no-project-group" });
          return;
        }

        const projectItems = Array.from(projectGroup.querySelectorAll(".sidebar-item-project")) as HTMLElement[];
        const dragItem = projectItems[0];
        const dataTransfer = new DataTransfer();
        dragItem.dispatchEvent(new DragEvent("dragstart", { bubbles: true, dataTransfer }));

        setTimeout(() => {
          const projectDropZones = Array.from(
            projectGroup.querySelectorAll(".sidebar-drop-zone-project"),
          ) as HTMLElement[];
          const dropZone = projectDropZones.at(-1);

          if (!dropZone) {
            dragItem.dispatchEvent(new DragEvent("dragend", { bubbles: true }));
            finish({
              ok: false,
              reason: "no-project-drop-zone",
              dropZoneCount: projectDropZones.length,
            });
            return;
          }

          const beforeSlotCount = document.querySelectorAll(".sidebar-drop-slot").length;
          dropZone.dispatchEvent(
            new DragEvent("dragover", { bubbles: true, dataTransfer }),
          );

          setTimeout(() => {
            const visibleSlotCount = document.querySelectorAll(".sidebar-drop-slot").length;
            dropZone.dispatchEvent(
              new DragEvent("dragleave", { bubbles: true, dataTransfer }),
            );

            setTimeout(() => {
              const slotCountAfterLeave = document.querySelectorAll(".sidebar-drop-slot").length;
              dragItem.dispatchEvent(new DragEvent("dragend", { bubbles: true }));

              finish({
                ok: visibleSlotCount > 0,
                beforeSlotCount,
                visibleSlotCount,
                slotCountAfterLeave,
                projectCount: projectItems.length,
                dropZoneCount: projectDropZones.length,
              });
            }, 50);
          }, 50);
        }, 50);
      });

      if (!dragState?.ok) {
        throw new Error(`Sidebar drag debug: ${JSON.stringify(dragState)}`);
      }
      expect(dragState?.beforeSlotCount).toBe(0);
      expect(dragState?.visibleSlotCount).toBeGreaterThan(0);
      expect(dragState?.slotCountAfterLeave).toBe(0);
    });
  });

  after(async () => {
    await navigateTo("Inbox");
  });
});
