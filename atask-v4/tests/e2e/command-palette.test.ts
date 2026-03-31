import {
  waitForAppReady,
  navigateTo,
  pressKeys,
  isCommandPaletteOpen,
  getViewTitle,
  elementExists,
} from "./helpers";

describe("Command Palette", () => {
  before(async () => {
    await waitForAppReady();
    await navigateTo("Inbox");
  });

  it("should open with ⇧⌘O", async () => {
    await pressKeys("O", true, true);
    expect(await isCommandPaletteOpen()).toBe(true);
  });

  it("should show command items", async () => {
    expect(await elementExists(".cmd-item")).toBe(true);
  });

  it("should close with Escape", async () => {
    // Dispatch Escape on the cmd-input (where the keydown handler lives)
    await browser.execute(() => {
      const input = document.querySelector(".cmd-input");
      if (input) {
        input.dispatchEvent(
          new KeyboardEvent("keydown", { key: "Escape", code: "Escape", bubbles: true }),
        );
      }
    });
    await browser.pause(300);
    expect(await isCommandPaletteOpen()).toBe(false);
  });

  it("should navigate via command palette", async () => {
    await pressKeys("O", true, true);
    await browser.pause(200);

    // Type "logbook" using the native value setter trick for React
    await browser.execute(() => {
      const input = document.querySelector(".cmd-input") as HTMLInputElement;
      if (input) {
        const nativeSet = Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value")?.set;
        if (nativeSet) {
          nativeSet.call(input, "logbook");
          input.dispatchEvent(new Event("input", { bubbles: true }));
        }
      }
    });
    await browser.pause(300);

    // Press Enter on the input to execute the selected command
    await browser.execute(() => {
      const input = document.querySelector(".cmd-input") as HTMLInputElement;
      if (input) {
        input.dispatchEvent(
          new KeyboardEvent("keydown", { key: "Enter", code: "Enter", bubbles: true }),
        );
      }
    });
    await browser.pause(500);

    expect(await getViewTitle()).toContain("Logbook");
    expect(await isCommandPaletteOpen()).toBe(false);
  });
});
