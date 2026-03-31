import { useState, useCallback } from "react";
import { useStore } from "@nanostores/react";
import {
  $activeView,
  $tasks,
  useActiveProjects,
  useActiveAreas,
  completeProject,
  reopenProject,
  deleteProject,
  moveProjectToArea,
  createProject,
  createArea,
  deleteArea,
  toggleAreaArchived,
  updateArea,
  updateProject,
  updateTask,
} from "../store/index";
import type { ActiveView, Project, Area } from "../types";
import ContextMenu, { type MenuItem } from "./ContextMenu";

// --- SVG Icons ---

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

// --- Nav Item ---

interface NavItemProps {
  view: ActiveView;
  label: string;
  icon: React.ReactNode;
  badge?: number;
  activeView: ActiveView;
  onClick: (view: ActiveView) => void;
  onTaskDrop?: (taskId: string) => void;
}

function NavItem({ view, label, icon, badge, activeView, onClick, onTaskDrop }: NavItemProps) {
  const [isDragTarget, setIsDragTarget] = useState(false);

  return (
    <div
      className={`sidebar-item${activeView === view ? " active" : ""}`}
      style={{ background: isDragTarget ? "var(--accent-subtle)" : undefined }}
      onClick={() => onClick(view)}
      onDragOver={onTaskDrop ? (e) => {
        e.preventDefault();
        e.dataTransfer.dropEffect = "move";
        setIsDragTarget(true);
      } : undefined}
      onDragLeave={onTaskDrop ? () => setIsDragTarget(false) : undefined}
      onDrop={onTaskDrop ? (e) => {
        e.preventDefault();
        setIsDragTarget(false);
        const taskId = e.dataTransfer.getData("text/plain");
        if (taskId) onTaskDrop(taskId);
      } : undefined}
    >
      <span className="sidebar-icon">{icon}</span>
      <span>{label}</span>
      {badge != null && badge > 0 && <span className="sidebar-badge">{badge}</span>}
    </div>
  );
}

// --- Project Item ---

interface ProjectItemProps {
  project: Project;
  badge: number;
  activeView: ActiveView;
  onClick: (view: ActiveView) => void;
  onContextMenu: (e: React.MouseEvent, project: Project) => void;
  isRenaming: boolean;
  renamingValue: string;
  onRenamingValueChange: (value: string) => void;
  onRenameCommit: () => void;
  onRenameCancel: () => void;
}

function ProjectItem({ project, badge, activeView, onClick, onContextMenu, isRenaming, renamingValue, onRenamingValueChange, onRenameCommit, onRenameCancel }: ProjectItemProps) {
  const view: ActiveView = `project-${project.id}`;
  const [isDragTarget, setIsDragTarget] = useState(false);

  return (
    <div
      className={`sidebar-item${activeView === view ? " active" : ""}`}
      style={{
        paddingLeft: "var(--sp-6)",
        background: isDragTarget ? "var(--accent-subtle)" : undefined,
      }}
      onClick={() => onClick(view)}
      onContextMenu={(e) => onContextMenu(e, project)}
      onDragOver={(e) => {
        e.preventDefault();
        e.dataTransfer.dropEffect = "move";
        setIsDragTarget(true);
      }}
      onDragLeave={() => setIsDragTarget(false)}
      onDrop={(e) => {
        e.preventDefault();
        setIsDragTarget(false);
        const taskId = e.dataTransfer.getData("text/plain");
        if (taskId) {
          updateTask({ id: taskId, projectId: project.id });
        }
      }}
    >
      <span
        className="sidebar-dot"
        style={{ background: project.color || "var(--accent)" }}
      />
      {isRenaming ? (
        <input
          autoFocus
          value={renamingValue}
          onChange={(e) => onRenamingValueChange(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter") {
              onRenameCommit();
            } else if (e.key === "Escape") {
              onRenameCancel();
            }
          }}
          onBlur={onRenameCommit}
          onClick={(e) => e.stopPropagation()}
          style={{
            flex: 1,
            background: "transparent",
            border: "none",
            borderBottom: "1px solid var(--accent)",
            outline: "none",
            fontSize: "inherit",
            color: "inherit",
            padding: "0",
          }}
        />
      ) : (
        <span>{project.title}</span>
      )}
      {badge > 0 && !isRenaming && <span className="sidebar-badge">{badge}</span>}
    </div>
  );
}

// --- Context menu state ---

type ContextMenuState =
  | { kind: "project"; project: Project; position: { x: number; y: number } }
  | { kind: "area"; area: Area; position: { x: number; y: number } }
  | null;

// --- Sidebar ---

export default function Sidebar() {
  const activeView = useStore($activeView);
  const tasks = useStore($tasks);
  const setActiveView = (v: ActiveView) => $activeView.set(v);

  const projects = useActiveProjects();
  const areas = useActiveAreas();

  const [contextMenu, setContextMenu] = useState<ContextMenuState>(null);
  const [renamingAreaId, setRenamingAreaId] = useState<string | null>(null);
  const [renamingAreaValue, setRenamingAreaValue] = useState("");
  const [renamingProjectId, setRenamingProjectId] = useState<string | null>(null);
  const [renamingProjectValue, setRenamingProjectValue] = useState("");

  const closeContextMenu = useCallback(() => setContextMenu(null), []);

  // --- Context menu handlers ---

  const handleProjectContextMenu = useCallback(
    (e: React.MouseEvent, project: Project) => {
      e.preventDefault();
      e.stopPropagation();
      setContextMenu({ kind: "project", project, position: { x: e.clientX, y: e.clientY } });
    },
    [],
  );

  const handleAreaContextMenu = useCallback(
    (e: React.MouseEvent, area: Area) => {
      e.preventDefault();
      e.stopPropagation();
      setContextMenu({ kind: "area", area, position: { x: e.clientX, y: e.clientY } });
    },
    [],
  );

  // Build project context menu items
  const buildProjectMenuItems = useCallback(
    (project: Project): MenuItem[] => {
      const isCompleted = project.status !== 0;
      const items: MenuItem[] = [
        {
          label: "Rename",
          onClick: () => {
            setRenamingProjectId(project.id);
            setRenamingProjectValue(project.title);
          },
        },
        {
          label: isCompleted ? "Reopen" : "Complete",
          onClick: () => {
            if (isCompleted) {
              reopenProject(project.id);
            } else {
              completeProject(project.id);
            }
          },
        },
        { separator: true },
      ];

      // "Move to Area" entries
      items.push({
        label: "No Area",
        onClick: () => moveProjectToArea(project.id, null),
        disabled: project.areaId === null,
      });
      for (const area of areas) {
        items.push({
          label: area.title,
          onClick: () => moveProjectToArea(project.id, area.id),
          disabled: project.areaId === area.id,
        });
      }

      items.push({ separator: true });
      items.push({
        label: "Delete",
        danger: true,
        onClick: () => deleteProject(project.id),
      });

      return items;
    },
    [areas],
  );

  // Build area context menu items
  const buildAreaMenuItems = useCallback(
    (area: Area): MenuItem[] => [
      {
        label: "Rename",
        onClick: () => {
          setRenamingAreaId(area.id);
          setRenamingAreaValue(area.title);
        },
      },
      {
        label: area.archived ? "Unarchive" : "Archive",
        onClick: () => toggleAreaArchived(area.id),
      },
      { separator: true },
      {
        label: "Delete",
        danger: true,
        onClick: () => deleteArea(area.id),
      },
    ],
    [],
  );

  const commitProjectRename = useCallback(() => {
    if (renamingProjectId) {
      const title = renamingProjectValue.trim();
      if (title) updateProject({ id: renamingProjectId, title });
      setRenamingProjectId(null);
    }
  }, [renamingProjectId, renamingProjectValue]);

  const cancelProjectRename = useCallback(() => {
    setRenamingProjectId(null);
  }, []);

  // Compute counts
  const inboxCount = tasks.filter((t) => t.schedule === 0 && t.status === 0).length;
  const todayCount = tasks.filter((t) => t.schedule === 1 && t.status === 0).length;

  // Task counts per project (open tasks only)
  const projectTaskCounts = new Map<string, number>();
  for (const t of tasks) {
    if (t.projectId && t.status === 0) {
      projectTaskCounts.set(t.projectId, (projectTaskCounts.get(t.projectId) ?? 0) + 1);
    }
  }

  // Group projects by area
  const projectsByArea = new Map<string | null, Project[]>();
  for (const p of projects) {
    const key = p.areaId;
    const list = projectsByArea.get(key) ?? [];
    list.push(p);
    projectsByArea.set(key, list);
  }

  // Derive context menu items
  const contextMenuItems: MenuItem[] =
    contextMenu?.kind === "project"
      ? buildProjectMenuItems(contextMenu.project)
      : contextMenu?.kind === "area"
        ? buildAreaMenuItems(contextMenu.area)
        : [];

  return (
    <div className="sidebar">
      <div className="sidebar-toolbar" data-tauri-drag-region>
      </div>

      {/* Nav group */}
      <div className="sidebar-group">
        <NavItem
          view="inbox"
          label="Inbox"
          icon={<InboxIcon />}
          badge={inboxCount}
          activeView={activeView}
          onClick={setActiveView}
          onTaskDrop={(taskId) => updateTask({ id: taskId, schedule: 0, startDate: null, timeSlot: null })}
        />
        <NavItem
          view="today"
          label="Today"
          icon={<TodayIcon />}
          badge={todayCount}
          activeView={activeView}
          onClick={setActiveView}
          onTaskDrop={(taskId) => {
            const today = new Date().toISOString().slice(0, 10);
            updateTask({ id: taskId, schedule: 1, startDate: today });
          }}
        />
        <NavItem
          view="upcoming"
          label="Upcoming"
          icon={<UpcomingIcon />}
          activeView={activeView}
          onClick={setActiveView}
        />
        <NavItem
          view="someday"
          label="Someday"
          icon={<SomedayIcon />}
          activeView={activeView}
          onClick={setActiveView}
          onTaskDrop={(taskId) => updateTask({ id: taskId, schedule: 2 })}
        />
        <NavItem
          view="logbook"
          label="Logbook"
          icon={<LogbookIcon />}
          activeView={activeView}
          onClick={setActiveView}
        />
      </div>

      <div className="sidebar-separator" />

      {/* Areas with their projects */}
      {areas.map((area) => {
        const areaProjects = projectsByArea.get(area.id) ?? [];
        return (
          <div className="sidebar-group" key={area.id}>
            {renamingAreaId === area.id ? (
              <div style={{ padding: "var(--sp-1) var(--sp-3)" }}>
                <input
                  autoFocus
                  value={renamingAreaValue}
                  onChange={(e) => setRenamingAreaValue(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === "Enter") {
                      const title = renamingAreaValue.trim();
                      if (title) updateArea({ id: area.id, title });
                      setRenamingAreaId(null);
                    } else if (e.key === "Escape") {
                      setRenamingAreaId(null);
                    }
                  }}
                  onBlur={() => {
                    const title = renamingAreaValue.trim();
                    if (title) updateArea({ id: area.id, title });
                    setRenamingAreaId(null);
                  }}
                  style={{
                    width: "100%",
                    background: "transparent",
                    border: "none",
                    borderBottom: "1px solid var(--accent)",
                    outline: "none",
                    fontSize: "var(--text-xs)",
                    fontWeight: 700,
                    color: "var(--ink-tertiary)",
                    textTransform: "uppercase",
                    letterSpacing: "0.5px",
                    padding: "var(--sp-1) 0",
                  }}
                />
              </div>
            ) : (
            <div
              className={`sidebar-group-label${activeView === `area-${area.id}` ? " active" : ""}`}
              onClick={() => setActiveView(`area-${area.id}`)}
              onContextMenu={(e) => handleAreaContextMenu(e, area)}
              style={{ display: "flex", alignItems: "center", cursor: "pointer" }}
            >
              <span style={{ flex: 1 }}>{area.title}</span>
              <button
                className="sidebar-add-btn"
                title={`Add project to ${area.title}`}
                onClick={async (e) => {
                  e.stopPropagation();
                  const project = await createProject({ title: "New Project", areaId: area.id });
                  if (project) {
                    setRenamingProjectId(project.id);
                    setRenamingProjectValue("New Project");
                  }
                }}
                style={{
                  background: "none",
                  border: "none",
                  color: "var(--ink-tertiary)",
                  cursor: "pointer",
                  fontSize: "var(--text-base)",
                  lineHeight: 1,
                  padding: "0 var(--sp-1)",
                  borderRadius: "var(--radius-sm)",
                }}
              >
                +
              </button>
            </div>
            )}
            {areaProjects.map((p) => (
              <ProjectItem
                key={p.id}
                project={p}
                badge={projectTaskCounts.get(p.id) ?? 0}
                activeView={activeView}
                onClick={setActiveView}
                onContextMenu={handleProjectContextMenu}
                isRenaming={renamingProjectId === p.id}
                renamingValue={renamingProjectValue}
                onRenamingValueChange={setRenamingProjectValue}
                onRenameCommit={commitProjectRename}
                onRenameCancel={cancelProjectRename}
              />
            ))}
          </div>
        );
      })}

      {/* Standalone projects (no area) — same visual style, no heading */}
      {(() => {
        const standalone = projectsByArea.get(null) ?? [];
        if (standalone.length === 0) return null;
        return (
          <div className="sidebar-group">
            {standalone.map((p) => (
              <ProjectItem
                key={p.id}
                project={p}
                badge={projectTaskCounts.get(p.id) ?? 0}
                activeView={activeView}
                onClick={setActiveView}
                onContextMenu={handleProjectContextMenu}
                isRenaming={renamingProjectId === p.id}
                renamingValue={renamingProjectValue}
                onRenamingValueChange={setRenamingProjectValue}
                onRenameCommit={commitProjectRename}
                onRenameCancel={cancelProjectRename}
              />
            ))}
          </div>
        );
      })()}

      {/* + New Area — creates area then enters rename mode */}
      <div style={{ padding: "var(--sp-1) var(--sp-3)" }}>
        <div
          className="sidebar-item"
          onClick={async () => {
            const area = await createArea({ title: "New Area" });
            if (area) {
              setRenamingAreaId(area.id);
              setRenamingAreaValue("New Area");
            }
          }}
          style={{ color: "var(--ink-tertiary)" }}
        >
          <span className="sidebar-icon" style={{ fontSize: "var(--text-base)" }}>+</span>
          <span>New Area</span>
        </div>
      </div>

      {/* Settings — pinned to bottom */}
      <div style={{ marginTop: 'auto', padding: 'var(--sp-2) var(--sp-3)', borderTop: '1px solid var(--separator)' }}>
        <div
          className={`sidebar-item${activeView === 'settings' ? ' active' : ''}`}
          onClick={() => setActiveView('settings')}
        >
          <span className="sidebar-icon">
            <svg viewBox="0 0 16 16" style={{ width: 16, height: 16 }}>
              <circle cx="8" cy="8" r="2.5" fill="none" stroke="currentColor" strokeWidth="1.5" />
              <path d="M8 1.5v2M8 12.5v2M1.5 8h2M12.5 8h2M3.2 3.2l1.4 1.4M11.4 11.4l1.4 1.4M3.2 12.8l1.4-1.4M11.4 4.6l1.4-1.4" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
            </svg>
          </span>
          <span>Settings</span>
        </div>
      </div>

      {/* Context menu */}
      {contextMenu && (
        <ContextMenu
          items={contextMenuItems}
          position={contextMenu.position}
          onClose={closeContextMenu}
        />
      )}
    </div>
  );
}
