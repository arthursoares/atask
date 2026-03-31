import {
  waitForAppReady,
  navigateTo,
  createTaskViaUI,
  clickTask,
  isDetailPanelVisible,
  clickDetailField,
  isWhenPickerVisible,
  getDetailFieldValue,
  pressKeys,
  getTaskTitles,
} from "./helpers";

describe("Calendar Date Selection", () => {
  before(async () => {
    await waitForAppReady();
    await navigateTo("Inbox");
  });

  it("should create a task for date testing", async () => {
    await createTaskViaUI("Date Test Task");
    const titles = await getTaskTitles();
    expect(titles).toContain("Date Test Task");
  });

  it("should open WhenPicker and see calendar", async () => {
    await clickTask("Date Test Task");
    expect(await isDetailPanelVisible()).toBe(true);

    await clickDetailField("Schedule");
    expect(await isWhenPickerVisible()).toBe(true);

    // Verify calendar grid exists
    const hasCalendar = await browser.execute(() => {
      return document.querySelector(".when-cal") !== null;
    });
    expect(hasCalendar).toBe(true);
  });

  it("should show day labels (Mo-Su)", async () => {
    const labels = await browser.execute(() => {
      const header = document.querySelector(".when-cal-header");
      return header?.textContent ?? "";
    });
    expect(labels).toContain("Mo");
    expect(labels).toContain("Fr");
    expect(labels).toContain("Su");
  });

  it("should highlight today in the calendar", async () => {
    const hasToday = await browser.execute(() => {
      return document.querySelector(".when-cal-day.today-cal") !== null;
    });
    expect(hasToday).toBe(true);
  });

  it("should select a future date from the calendar", async () => {
    // Find a day in the future and click it
    const clicked = await browser.execute(() => {
      const today = new Date().getDate();
      const days = document.querySelectorAll(".when-cal-day:not(.empty)");
      // Find a day > today (or wrap to next available)
      for (const day of days) {
        const num = parseInt(day.textContent ?? "0", 10);
        if (num > today && num <= today + 7) {
          (day as HTMLElement).click();
          return num;
        }
      }
      // If near end of month, just pick the last available day
      for (let i = days.length - 1; i >= 0; i--) {
        const num = parseInt(days[i].textContent ?? "0", 10);
        if (num > today) {
          (days[i] as HTMLElement).click();
          return num;
        }
      }
      return 0;
    });

    await browser.pause(300);

    if (clicked > 0) {
      // The task should now have a start date set
      // Check Start Date field in detail panel
      const startDate = await getDetailFieldValue("Start Date");
      expect(startDate).not.toBe("None");
    }
  });

  it("should show task in Upcoming view after setting future date", async () => {
    // Close detail panel
    await pressKeys("Escape");
    await browser.pause(200);

    await navigateTo("Upcoming");
    await browser.pause(500);

    const titles = await getTaskTitles();
    // The task should be in Upcoming since we set a future start date
    expect(titles).toContain("Date Test Task");
  });

  after(async () => {
    await navigateTo("Inbox");
  });
});
