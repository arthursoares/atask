import { useEffect } from "react";
import { useStore } from "@nanostores/react";
import { loadAll, $showSidebar, $selectedTaskId, $activeView, $selectedTaskIds } from "./store/index";
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
import useKeyboard from "./hooks/useKeyboard";
import useSync from "./hooks/useSync";

function App() {
  const showSidebar = useStore($showSidebar);
  const selectedTaskId = useStore($selectedTaskId);
  const activeView = useStore($activeView);
  const selectedTaskIds = useStore($selectedTaskIds);

  useEffect(() => {
    loadAll();
  }, []);

  useKeyboard();
  useSync();

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
    </>
  );
}

export default App;
