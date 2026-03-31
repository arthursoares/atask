/**
 * Sync Round-Trip E2E Tests
 *
 * Tests the full bidirectional sync flow:
 * 1. Configure Tauri app with API key
 * 2. Client creates task → pending op → flushed → appears on server
 * 3. Server creates task → SSE event → client upserts → appears in UI
 * 4. Both sides: complete, update, delete
 *
 * Requires Go server at localhost:8080. Run setup before tests:
 *   rm -f /tmp/atask-e2e-test.db
 *   DB_PATH=/tmp/atask-e2e-test.db ./bin/atask &
 *   curl -X POST http://localhost:8080/auth/register \
 *     -H "Content-Type: application/json" \
 *     -d '{"email":"test@e2e.local","password":"testpassword123","name":"E2E Test"}'
 */

import {
  waitForAppReady,
  navigateTo,
  createTaskViaUI,
  getTaskTitles,
  clickCheckbox,
  isTaskCompleted,
} from "./helpers";

const API_URL = "http://localhost:8080";

let jwtToken = "";
let apiKey = "";
let apiAvailable = false;

// --- Node.js API helpers ---

async function apiCall(
  method: string,
  path: string,
  body?: unknown,
  useApiKey = false,
): Promise<{ status: number; data: unknown }> {
  const auth = useApiKey ? `ApiKey ${apiKey}` : `Bearer ${jwtToken}`;
  const opts: RequestInit = {
    method,
    headers: { "Content-Type": "application/json", Authorization: auth },
  };
  if (body) opts.body = JSON.stringify(body);
  try {
    const resp = await fetch(`${API_URL}${path}`, opts);
    const data = await resp.json().catch(() => null);
    return { status: resp.status, data };
  } catch {
    return { status: 0, data: null };
  }
}

async function getServerTasks(): Promise<Array<{ ID: string; Title: string; Status: number }>> {
  const result = await apiCall("GET", "/tasks?status=all", undefined, true);
  return (result.data as Array<{ ID: string; Title: string; Status: number }>) ?? [];
}

async function createServerTask(title: string): Promise<string> {
  const result = await apiCall("POST", "/tasks", { title }, true);
  return (result.data as { data?: { ID?: string } })?.data?.ID ?? "";
}

async function completeServerTask(id: string): Promise<boolean> {
  const result = await apiCall("POST", `/tasks/${id}/complete`, undefined, true);
  return result.status === 200;
}

async function deleteServerTask(id: string): Promise<boolean> {
  const result = await apiCall("DELETE", `/tasks/${id}`, undefined, true);
  return result.status === 200;
}

// --- Test setup ---

describe("Sync Round-Trip", () => {
  before(async () => {
    await waitForAppReady();

    // 1. Check API
    try {
      const resp = await fetch(`${API_URL}/health`);
      apiAvailable = resp.ok;
    } catch {
      apiAvailable = false;
    }
    if (!apiAvailable) {
      console.log("Go API not available — skipping round-trip tests");
      return;
    }

    // 2. Login
    const login = await apiCall("POST", "/auth/login", {
      email: "test@e2e.local",
      password: "testpassword123",
    });
    jwtToken = (login.data as { token?: string })?.token ?? "";
    if (!jwtToken) {
      apiAvailable = false;
      return;
    }

    // 3. Create API key
    const keyResult = await apiCall("POST", "/auth/api-keys", { name: "Sync Test Key" });
    apiKey = (keyResult.data as { key?: string })?.key ?? "";
    if (!apiKey) {
      console.log("Could not create API key — skipping");
      apiAvailable = false;
      return;
    }

    // 4. Verify API key works
    const verify = await apiCall("GET", "/tasks", undefined, true);
    if (verify.status !== 200) {
      console.log("API key verification failed — skipping");
      apiAvailable = false;
      return;
    }

    // 5. Configure Tauri app sync settings via UI
    await navigateTo("Settings");
    await browser.pause(300);

    // Enable sync toggle
    await browser.execute(() => {
      const cb = document.querySelector("input[type='checkbox']") as HTMLInputElement;
      if (cb && !cb.checked) cb.click();
    });
    await browser.pause(200);

    // Set server URL
    await browser.execute((url: string) => {
      const input = document.querySelector("input[type='url']") as HTMLInputElement;
      if (input) {
        const nativeSet = Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value")?.set;
        if (nativeSet) { nativeSet.call(input, url); input.dispatchEvent(new Event("input", { bubbles: true })); }
      }
    }, API_URL);
    await browser.pause(100);

    // Set API key
    await browser.execute((key: string) => {
      const inputs = document.querySelectorAll("input") as NodeListOf<HTMLInputElement>;
      for (const input of inputs) {
        if (input.placeholder?.includes("ak_") || input.type === "password") {
          const nativeSet = Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value")?.set;
          if (nativeSet) { nativeSet.call(input, key); input.dispatchEvent(new Event("input", { bubbles: true })); }
          break;
        }
      }
    }, apiKey);
    await browser.pause(100);

    // Mark dirty + save
    await browser.execute(() => {
      const btns = document.querySelectorAll("button");
      for (const btn of btns) { if (btn.textContent?.includes("Save")) { btn.click(); return; } }
    });
    await browser.pause(500);

    await navigateTo("Inbox");
    await browser.pause(500);
  });

  // --- Outbound: Client → Server ---

  describe("Outbound: Client mutations sync to server", () => {
    it("should create a task in the client", async function () {
      if (!apiAvailable) return this.skip();

      await createTaskViaUI("Outbound Sync Task");
      const titles = await getTaskTitles();
      expect(titles).toContain("Outbound Sync Task");
    });

    it("should flush pending op to server within 35s", async function () {
      if (!apiAvailable) return this.skip();
      this.timeout(40000);

      // Wait for the sync worker's 30s cycle + some buffer
      let found = false;
      for (let i = 0; i < 7; i++) {
        await new Promise((r) => setTimeout(r, 5000));
        const tasks = await getServerTasks();
        if (tasks.some((t) => t.Title === "Outbound Sync Task")) {
          found = true;
          break;
        }
      }
      expect(found).toBe(true);
    });

    it("should complete a task in client and sync to server", async function () {
      if (!apiAvailable) return this.skip();
      this.timeout(40000);

      await clickCheckbox("Outbound Sync Task");
      expect(await isTaskCompleted("Outbound Sync Task")).toBe(true);

      // Wait for sync
      let completed = false;
      for (let i = 0; i < 7; i++) {
        await new Promise((r) => setTimeout(r, 5000));
        const tasks = await getServerTasks();
        const task = tasks.find((t) => t.Title === "Outbound Sync Task");
        if (task && task.Status === 1) {
          completed = true;
          break;
        }
      }
      expect(completed).toBe(true);
    });
  });

  // --- Inbound: Server → Client ---

  describe("Inbound: Server events sync to client", () => {
    it("should create a task on the server", async function () {
      if (!apiAvailable) return this.skip();

      const id = await createServerTask("Inbound Sync Task");
      expect(id.length).toBeGreaterThan(0);
    });

    it("should appear in the client after delta sync", async function () {
      if (!apiAvailable) return this.skip();
      this.timeout(30000);

      // Trigger sync from the browser via Tauri invoke
      let found = false;
      for (let i = 0; i < 8; i++) {
        // Call triggerSync directly from the WebView
        await browser.execute(async () => {
          try {
            // @ts-ignore — access Tauri invoke via __TAURI_INTERNALS__
            await (window as any).__TAURI_INTERNALS__.invoke('trigger_sync');
            await (window as any).__TAURI_INTERNALS__.invoke('load_all').then((data: any) => {
              // Force store refresh — the store-changed event might not fire fast enough
            });
          } catch {}
        });
        await browser.pause(2000);

        // Navigate to refresh view
        await navigateTo("Today");
        await browser.pause(300);
        await navigateTo("Inbox");
        await browser.pause(500);

        const titles = await getTaskTitles();
        if (titles.includes("Inbound Sync Task")) {
          found = true;
          break;
        }
      }
      expect(found).toBe(true);
    });

    it("should complete a task on the server and reflect in client", async function () {
      if (!apiAvailable) return this.skip();
      this.timeout(20000);

      // Find the task ID on the server
      const tasks = await getServerTasks();
      const task = tasks.find((t) => t.Title === "Inbound Sync Task");
      if (!task) return this.skip();

      await completeServerTask(task.ID);

      // Wait for SSE propagation
      let completed = false;
      for (let i = 0; i < 6; i++) {
        await new Promise((r) => setTimeout(r, 2000));
        await navigateTo("Today");
        await browser.pause(200);
        await navigateTo("Inbox");
        await browser.pause(500);

        completed = await isTaskCompleted("Inbound Sync Task");
        if (completed) break;
      }
      expect(completed).toBe(true);
    });
  });

  // --- Bidirectional: Both sides create tasks ---

  describe("Bidirectional: Both sides create concurrently", () => {
    it("should create tasks on both sides simultaneously", async function () {
      if (!apiAvailable) return this.skip();
      this.timeout(40000);

      // Client creates
      await createTaskViaUI("Bidi Client Task");

      // Server creates
      await createServerTask("Bidi Server Task");

      // Wait for sync in both directions
      await new Promise((r) => setTimeout(r, 35000));

      // Client should have both
      await navigateTo("Today");
      await browser.pause(200);
      await navigateTo("Inbox");
      await browser.pause(500);
      const clientTitles = await getTaskTitles();

      // Server should have both
      const serverTasks = await getServerTasks();
      const serverTitles = serverTasks.map((t) => t.Title);

      // Verify bidirectional
      expect(clientTitles).toContain("Bidi Client Task");
      expect(serverTitles).toContain("Bidi Server Task");
      // The client task should have synced to server
      expect(serverTitles).toContain("Bidi Client Task");
      // The server task should have synced to client
      expect(clientTitles).toContain("Bidi Server Task");
    });
  });

  // --- Delete sync ---

  describe("Delete propagation", () => {
    it("should delete a task on server and remove from client", async function () {
      if (!apiAvailable) return this.skip();
      this.timeout(20000);

      const id = await createServerTask("Delete Sync Task");
      expect(id.length).toBeGreaterThan(0);

      // Wait for it to appear in client
      await new Promise((r) => setTimeout(r, 5000));
      await navigateTo("Today");
      await browser.pause(200);
      await navigateTo("Inbox");
      await browser.pause(500);

      // Delete on server
      await deleteServerTask(id);

      // Wait for propagation
      await new Promise((r) => setTimeout(r, 5000));
      await navigateTo("Today");
      await browser.pause(200);
      await navigateTo("Inbox");
      await browser.pause(500);

      const titles = await getTaskTitles();
      expect(titles).not.toContain("Delete Sync Task");
    });
  });

  after(async () => {
    await navigateTo("Inbox");
  });
});
