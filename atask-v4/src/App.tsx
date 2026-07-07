import { useEffect, useState } from "react";
import { useStore } from "@nanostores/react";
import {
  loadAll,
  showErrorToast,
  $showSidebar,
  $selectedTaskId,
  $activeView,
  $selectedTaskIds,
  $authState,
} from "./store/index";
import { refreshOnLaunch } from "./hooks/useTauri";
import Sidebar from "./components/Sidebar";
import Toolbar from "./components/Toolbar";
import InboxView from "./views/InboxView";
import TodayView from "./views/TodayView";
import UpcomingView from "./views/UpcomingView";
import SomedayView from "./views/SomedayView";
import LogbookView from "./views/LogbookView";
import ProjectView from "./views/ProjectView";
import AreaView from "./views/AreaView";
import SettingsView from "./views/SettingsView";
import DetailPanel from "./components/DetailPanel";
import CommandPalette from "./components/CommandPalette";
import SearchOverlay from "./components/SearchOverlay";
import QuickMovePicker from "./components/QuickMovePicker";
import BulkActionBar from "./components/BulkActionBar";
import ShortcutsHelp from "./components/ShortcutsHelp";
import ToastHost from "./components/ToastHost";
import { Button } from "./ui";
import useKeyboard from "./hooks/useKeyboard";
import useSync from "./hooks/useSync";

function App() {
  const showSidebar = useStore($showSidebar);
  const selectedTaskId = useStore($selectedTaskId);
  const activeView = useStore($activeView);
  const selectedTaskIds = useStore($selectedTaskIds);
  const [loadState, setLoadState] = useState<"loading" | "ready" | "error">("loading");

  useEffect(() => {
    loadAll()
      .then(() => setLoadState("ready"))
      .catch((err) => {
        setLoadState("error");
        showErrorToast(`Couldn't load your data: ${String(err)}`);
      });
    // Re-derive the in-memory session from the keychain-stored token (if
    // any) on every app launch. Silently leaves $authState unauthenticated
    // if there's no prior session or the refresh fails.
    refreshOnLaunch()
      .then((state) => $authState.set(state))
      .catch(() => {});
  }, []);

  // Most mutations are fired from event handlers without awaiting; a failed
  // Tauri invoke would otherwise vanish into the console. Surface it and
  // reload state so the UI can't silently drift from the database.
  useEffect(() => {
    let reloadScheduled = false;
    const handler = (event: PromiseRejectionEvent) => {
      event.preventDefault();
      showErrorToast(`Something went wrong: ${String(event.reason)}`);
      if (!reloadScheduled) {
        reloadScheduled = true;
        void loadAll().finally(() => { reloadScheduled = false; });
      }
    };
    window.addEventListener("unhandledrejection", handler);
    return () => window.removeEventListener("unhandledrejection", handler);
  }, []);

  useKeyboard();
  useSync();

  if (loadState === "loading") {
    return (
      <div className="app-frame app-loading" aria-busy="true">
        <div className="app-loading-spinner" aria-hidden="true" />
        <span className="app-loading-text">Loading…</span>
      </div>
    );
  }

  if (loadState === "error") {
    return (
      <div className="app-frame app-loading">
        <span className="app-loading-text">Couldn't load your data.</span>
        <Button
          onClick={() => {
            setLoadState("loading");
            loadAll()
              .then(() => setLoadState("ready"))
              .catch(() => setLoadState("error"));
          }}
        >
          Retry
        </Button>
        <ToastHost />
      </div>
    );
  }

  return (
    <>
      <div className="app-frame">
        {showSidebar && <Sidebar />}
        <div className="app-main">
          <Toolbar />
          <div className="app-content">
            {activeView === "inbox" && <InboxView />}
            {activeView === "today" && <TodayView />}
            {activeView === "upcoming" && <UpcomingView />}
            {activeView === "someday" && <SomedayView />}
            {activeView === "logbook" && <LogbookView />}
            {activeView === "settings" && <SettingsView />}
            {activeView.startsWith("project-") && (
              <ProjectView projectId={activeView.slice("project-".length)} />
            )}
            {activeView.startsWith("area-") && (
              <AreaView areaId={activeView.slice("area-".length)} />
            )}
          </div>
        </div>
        {selectedTaskId && selectedTaskIds.size === 0 && <DetailPanel taskId={selectedTaskId} />}
      </div>
      <CommandPalette />
      <SearchOverlay />
      <QuickMovePicker />
      <BulkActionBar />
      <ShortcutsHelp />
      <ToastHost />
    </>
  );
}

export default App;
