import {
  waitForAppReady,
  resetDatabase,
  navigateTo,
  createTaskViaUI,
  clickTask,
  clickCheckbox,
} from "./helpers";

describe("Activities", () => {
  before(async () => {
    await waitForAppReady();
    await navigateTo("Inbox");
  });

  beforeEach(async () => {
    await resetDatabase();
    await waitForAppReady();
  });

  describe("Empty state", () => {
    it("should show 'No activity yet' for a new task", async () => {
      await createTaskViaUI("Activity Empty Test");
      await clickTask("Activity Empty Test");
      await browser.pause(300);

      const emptyText = await browser.execute(() => {
        const el = document.querySelector(".activity-empty");
        return el?.textContent ?? null;
      });
      expect(emptyText).toBe("No activity yet");
    });
  });

  describe("Mutation log", () => {
    it("should show 'Completed' activity after completing a task", async () => {
      await createTaskViaUI("Activity Mutation Test");
      await clickCheckbox("Activity Mutation Test");
      await browser.pause(500);

      // Navigate to Logbook where completed tasks live
      await navigateTo("Logbook");
      await browser.pause(300);

      // Click the completed task in logbook to open detail panel
      await browser.execute(() => {
        const rows = document.querySelectorAll(".logbook-row");
        for (const row of rows) {
          const title = row.querySelector(".task-title");
          if (title?.textContent?.trim() === "Activity Mutation Test") {
            (row as HTMLElement).click();
            return;
          }
        }
      });
      await browser.pause(500);

      // Verify a status_change activity with "Completed" text appears
      const hasMutationEntry = await browser.execute(() => {
        const entries = document.querySelectorAll(".activity-mutation-text");
        return Array.from(entries).some((el) =>
          el.textContent?.toLowerCase().includes("completed"),
        );
      });
      expect(hasMutationEntry).toBe(true);
    });
  });

  describe("Comments", () => {
    it("should add a comment via the input field", async () => {
      await createTaskViaUI("Activity Comment Test");
      await clickTask("Activity Comment Test");
      await browser.pause(300);

      // Type a comment using React's native value setter pattern
      await browser.execute(() => {
        const input = document.querySelector(".activity-comment-field") as HTMLInputElement;
        if (input) {
          const nativeSet = Object.getOwnPropertyDescriptor(
            HTMLInputElement.prototype,
            "value",
          )?.set;
          if (nativeSet) {
            nativeSet.call(input, "Hello from E2E");
            input.dispatchEvent(new Event("input", { bubbles: true }));
          }
        }
      });
      await browser.pause(100);

      // Press Enter to submit
      await browser.execute(() => {
        const input = document.querySelector(".activity-comment-field") as HTMLInputElement;
        if (input) {
          input.dispatchEvent(
            new KeyboardEvent("keydown", { key: "Enter", code: "Enter", bubbles: true }),
          );
        }
      });
      await browser.pause(500);

      // Verify the comment appears in the activity feed
      const commentText = await browser.execute(() => {
        const entries = document.querySelectorAll(".activity-entry .activity-text");
        return Array.from(entries)
          .map((el) => el.textContent ?? "")
          .filter((t) => t === "Hello from E2E");
      });
      expect(commentText.length).toBeGreaterThanOrEqual(1);

      // Verify the empty state is gone
      const emptyEl = await browser.execute(() => {
        return document.querySelector(".activity-empty");
      });
      expect(emptyEl).toBeNull();
    });
  });

  after(async () => {
    await navigateTo("Inbox");
  });
});
