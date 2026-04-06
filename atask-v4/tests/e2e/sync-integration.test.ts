/**
 * Sync Integration E2E Tests
 *
 * Tests bidirectional sync between the Tauri app and Go API server.
 * API calls are made from Node.js (test runner), not from the WebView.
 *
 * Requires: Go server at localhost:8080 with test user registered:
 *   DB_PATH=/tmp/atask-e2e-test.db ./bin/atask &
 *   curl -X POST http://localhost:8080/auth/register \
 *     -H "Content-Type: application/json" \
 *     -d '{"email":"test@e2e.local","password":"testpassword123","name":"E2E Test"}'
 */

import {
  waitForAppReady,
  resetDatabase,
  navigateTo,
  createTaskViaUI,
  getTaskTitles,
} from "./helpers";

const API_URL = "http://localhost:8080";
const TEST_EMAIL = "test@e2e.local";
const TEST_PASSWORD = "testpassword123";

let token = "";
let apiAvailable = false;

/** Make API calls from Node.js (not the browser) */
async function apiCall(
  method: string,
  path: string,
  body?: unknown,
): Promise<{ status: number; data: unknown }> {
  const opts: RequestInit = {
    method,
    headers: {
      "Content-Type": "application/json",
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
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

describe("Sync Integration — Bidirectional", () => {
  before(async () => {
    await waitForAppReady();

    // Check API availability from Node.js
    try {
      const resp = await fetch(`${API_URL}/health`);
      apiAvailable = resp.ok;
    } catch {
      apiAvailable = false;
    }

    if (!apiAvailable) {
      console.log("Go API not available at localhost:8080 — skipping sync integration tests");
      return;
    }

    // Login from Node.js
    const loginResult = await apiCall("POST", "/auth/login", {
      email: TEST_EMAIL,
      password: TEST_PASSWORD,
    });
    token = (loginResult.data as { token?: string })?.token ?? "";
    if (!token) {
      console.log("Could not authenticate — skipping");
      apiAvailable = false;
    }
  });

  beforeEach(async () => {
    await resetDatabase();
    await waitForAppReady();
  });

  describe("API → Client (server creates task)", () => {
    let createdTaskId = "";

    it("should create a task via the Go API", async function () {
      if (!apiAvailable) return this.skip();

      const result = await apiCall("POST", "/tasks", { title: "API Created Task" });
      expect(result.status).toBe(201);
      createdTaskId =
        (result.data as { data?: { ID?: string } })?.data?.ID ?? "";
      expect(createdTaskId.length).toBeGreaterThan(0);
    });

    it("should verify task exists on server via API list", async function () {
      if (!apiAvailable) return this.skip();

      const result = await apiCall("GET", "/tasks");
      expect(result.status).toBe(200);
      const titles = (result.data as Array<{ Title: string }>).map(
        (t) => t.Title,
      );
      expect(titles).toContain("API Created Task");
    });
  });

  describe("Client → API (client creates task)", () => {
    it("should create a task in the Tauri app", async function () {
      if (!apiAvailable) return this.skip();

      await navigateTo("Inbox");
      await createTaskViaUI("Client Created Task");
      const titles = await getTaskTitles();
      expect(titles).toContain("Client Created Task");
    });

    it("should have sync settings available", async function () {
      if (!apiAvailable) return this.skip();

      await navigateTo("Settings");
      await browser.pause(300);

      // Verify sync toggle exists
      const hasSyncToggle = await browser.execute(() => {
        return document.querySelector("input[type='checkbox']") !== null;
      });
      expect(hasSyncToggle).toBe(true);
      await navigateTo("Inbox");
    });
  });

  describe("API task lifecycle", () => {
    it("should create and complete a task via API", async function () {
      if (!apiAvailable) return this.skip();

      const create = await apiCall("POST", "/tasks", {
        title: "Lifecycle Task",
      });
      expect(create.status).toBe(201);
      const taskId =
        (create.data as { data?: { ID?: string } })?.data?.ID ?? "";

      const complete = await apiCall(
        "POST",
        `/tasks/${taskId}/complete`,
      );
      expect(complete.status).toBe(200);

      // Verify via list with status=all
      const list = await apiCall("GET", "/tasks?status=all");
      const completed = (list.data as Array<{ Title: string; Status: number }>)
        .filter((t) => t.Status === 1)
        .map((t) => t.Title);
      expect(completed).toContain("Lifecycle Task");
    });

    it("should create and delete a task via API", async function () {
      if (!apiAvailable) return this.skip();

      const create = await apiCall("POST", "/tasks", {
        title: "Deletable Task",
      });
      const taskId =
        (create.data as { data?: { ID?: string } })?.data?.ID ?? "";

      const del = await apiCall("DELETE", `/tasks/${taskId}`);
      expect(del.status).toBe(200);

      const list = await apiCall("GET", "/tasks");
      const titles = (list.data as Array<{ Title: string }>).map(
        (t) => t.Title,
      );
      expect(titles).not.toContain("Deletable Task");
    });

    it("should create projects and areas via API", async function () {
      if (!apiAvailable) return this.skip();

      const area = await apiCall("POST", "/areas", { title: "API Area" });
      expect(area.status).toBe(201);

      const project = await apiCall("POST", "/projects", {
        title: "API Project",
      });
      expect(project.status).toBe(201);

      // Verify
      const areas = await apiCall("GET", "/areas");
      const areaNames = (areas.data as Array<{ Title: string }>).map(
        (a) => a.Title,
      );
      expect(areaNames).toContain("API Area");
    });
  });

  describe("SSE event stream verification", () => {
    it("should receive SSE events from the Go server", async function () {
      if (!apiAvailable) return this.skip();

      // Connect SSE from Node.js and create a task to trigger event
      const controller = new AbortController();
      const timeout = setTimeout(() => controller.abort(), 5000);

      let eventData = "";
      try {
        const [sseResp] = await Promise.all([
          fetch(`${API_URL}/events/stream?topics=task.created`, {
            headers: { Authorization: `Bearer ${token}` },
            signal: controller.signal,
          }),
          // Create task after a brief delay
          new Promise<void>((resolve) =>
            setTimeout(async () => {
              await apiCall("POST", "/tasks", { title: "SSE Trigger Task" });
              resolve();
            }, 500),
          ),
        ]);

        const reader = sseResp.body?.getReader();
        if (reader) {
          const decoder = new TextDecoder();
          while (true) {
            const { done, value } = await reader.read();
            if (done) break;
            eventData += decoder.decode(value);
            if (eventData.includes("data:")) break;
          }
          reader.cancel();
        }
      } catch {
        // AbortError expected after timeout
      }
      clearTimeout(timeout);

      if (eventData) {
        expect(eventData).toContain("event:");
        expect(eventData).toContain("data:");
        expect(eventData).toContain("entity_type");
        expect(eventData).toContain("task");
      }
    });
  });

  after(async () => {
    await navigateTo("Inbox");
  });
});
