import { useStore } from "@nanostores/react";
import { $activeView, $projects, $areas, $showPalette, $showSearch, createTask } from "../store/index";
import SyncStatusIndicator from "./SyncStatusIndicator";

// --- Icon components matching the active view ---

function InboxIcon() {
  return (
    <svg viewBox="0 0 16 16">
      <rect x="2" y="3" width="12" height="10" rx="2" />
      <polyline points="2 8 6 8 7 10 9 10 10 8 14 8" />
    </svg>
  );
}

function TodayIcon() {
  return (
    <svg viewBox="0 0 16 16" fill="var(--today-star)" stroke="none">
      <polygon points="8 2 9.8 5.6 14 6.2 11 9 11.8 13 8 11.2 4.2 13 5 9 2 6.2 6.2 5.6" />
    </svg>
  );
}

function UpcomingIcon() {
  return (
    <svg viewBox="0 0 16 16">
      <rect x="2" y="3" width="12" height="11" rx="2" />
      <line x1="2" y1="7" x2="14" y2="7" />
      <line x1="5" y1="1" x2="5" y2="4" />
      <line x1="11" y1="1" x2="11" y2="4" />
    </svg>
  );
}

function SomedayIcon() {
  return (
    <svg viewBox="0 0 16 16" stroke="var(--someday-tint)">
      <circle cx="8" cy="8" r="5.5" />
      <line x1="8" y1="5" x2="8" y2="8" />
      <line x1="8" y1="8" x2="10.5" y2="10" />
    </svg>
  );
}

function LogbookIcon() {
  return (
    <svg viewBox="0 0 16 16">
      <path d="M4 2h8l1 4-5 3-5-3z" />
      <path d="M3 6v6c0 1 2 2 5 2s5-1 5-2V6" />
    </svg>
  );
}

function SearchIcon() {
  return (
    <svg viewBox="0 0 16 16">
      <circle cx="7" cy="7" r="4.5" />
      <line x1="10.2" y1="10.2" x2="14" y2="14" />
    </svg>
  );
}

function NewTaskIcon() {
  return (
    <svg viewBox="0 0 16 16">
      <line x1="8" y1="3" x2="8" y2="13" />
      <line x1="3" y1="8" x2="13" y2="8" />
    </svg>
  );
}

function CommandPaletteIcon() {
  return (
    <svg viewBox="0 0 16 16">
      <rect x="1" y="4" width="14" height="9" rx="2" />
      <text
        x="8"
        y="10.5"
        textAnchor="middle"
        fill="currentColor"
        stroke="none"
        fontSize="6"
        fontWeight="700"
        fontFamily="system-ui"
      >
        {"\u2318K"}
      </text>
    </svg>
  );
}

// --- View title config ---

interface ViewConfig {
  title: string;
  icon: React.ReactNode;
}

function getViewConfig(activeView: string, projectTitle?: string, areaTitle?: string): ViewConfig {
  switch (activeView) {
    case "inbox":
      return { title: "Inbox", icon: <InboxIcon /> };
    case "today":
      return { title: "Today", icon: <TodayIcon /> };
    case "upcoming":
      return { title: "Upcoming", icon: <UpcomingIcon /> };
    case "someday":
      return { title: "Someday", icon: <SomedayIcon /> };
    case "logbook":
      return { title: "Logbook", icon: <LogbookIcon /> };
    default:
      if (activeView.startsWith("project-")) {
        return { title: projectTitle ?? "Project", icon: null };
      }
      if (activeView.startsWith("area-")) {
        return { title: areaTitle ?? "Area", icon: null };
      }
      return { title: activeView, icon: null };
  }
}

function formatTodayDate(): string {
  return new Date().toLocaleDateString("en-US", {
    weekday: "long",
    month: "short",
    day: "numeric",
  });
}

// --- Toolbar ---

export default function Toolbar() {
  const activeView = useStore($activeView);
  const projects = useStore($projects);
  const areas = useStore($areas);
  const setShowPalette = (v: boolean) => $showPalette.set(v);

  // Resolve project/area title for view header
  let projectTitle: string | undefined;
  let areaTitle: string | undefined;
  if (activeView.startsWith("project-")) {
    const projectId = activeView.slice("project-".length);
    projectTitle = projects.find((p) => p.id === projectId)?.title;
  } else if (activeView.startsWith("area-")) {
    const areaId = activeView.slice("area-".length);
    areaTitle = areas.find((a) => a.id === areaId)?.title;
  }

  const config = getViewConfig(activeView, projectTitle, areaTitle);

  return (
    <div className="app-toolbar" data-tauri-drag-region>
      <div className="app-toolbar-left">
        <div className="app-view-title">
          {config.icon && <span className="sidebar-icon">{config.icon}</span>}
          {config.title}
        </div>
        {activeView === "today" && (
          <span className="toolbar-subtitle">{formatTodayDate()}</span>
        )}
      </div>
      <div className="app-toolbar-right">
        <SyncStatusIndicator />
        <button
          className="toolbar-btn"
          title="Search (⌘F)"
          aria-label="Search tasks"
          onClick={() => $showSearch.set(true)}
        >
          <SearchIcon aria-hidden="true" />
        </button>
        <button
          className="toolbar-btn"
          title="New Task (⌘N)"
          aria-label="Create new task"
          onClick={() => createTask("")}
        >
          <NewTaskIcon aria-hidden="true" />
        </button>
        <button
          className="toolbar-btn"
          title="Command Palette (⌘⇧P)"
          aria-label="Open command palette"
          onClick={() => setShowPalette(true)}
        >
          <CommandPaletteIcon aria-hidden="true" />
        </button>
      </div>
    </div>
  );
}
