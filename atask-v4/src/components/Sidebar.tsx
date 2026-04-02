import { Fragment, useState, useCallback } from "react";
import { useStore } from "@nanostores/react";
import {
  $activeView,
  $areas,
  $projects,
  $tasks,
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
  reorderProjects,
  reorderAreas,
} from "../store/index";
import type { ActiveView, Project, Area } from "../types";
import ContextMenu, { type MenuItem } from "./ContextMenu";
import { Button } from "../ui";
import { LogbookIcon, InboxIcon, SomedayIcon, TodayIcon, UpcomingIcon } from "./sidebar/SidebarIcons";
import {
  NavItem,
  ProjectItem,
  SidebarDropSlot,
  SidebarRenameField,
  SidebarRow,
  SIDEBAR_REORDER_MIME,
} from "./sidebar/SidebarParts";

// --- Context menu state ---

type ContextMenuState =
  | { kind: "project"; project: Project; position: { x: number; y: number } }
  | { kind: "area"; area: Area; position: { x: number; y: number } }
  | null;

type SidebarReorderDrag =
  | { kind: "area"; id: string }
  | { kind: "project"; id: string; areaId: string | null };

type SidebarInsertionTarget =
  | { kind: "area"; index: number }
  | { kind: "project"; areaId: string | null; index: number };

// --- Sidebar ---

export default function Sidebar() {
  const activeView = useStore($activeView);
  const allProjects = useStore($projects);
  const allAreas = useStore($areas);
  const tasks = useStore($tasks);
  const setActiveView = (v: ActiveView) => $activeView.set(v);

  const projects = [...allProjects].filter((project) => project.status === 0).sort((a, b) => a.index - b.index);
  const areas = [...allAreas].filter((area) => !area.archived).sort((a, b) => a.index - b.index);

  const [contextMenu, setContextMenu] = useState<ContextMenuState>(null);
  const [renamingAreaId, setRenamingAreaId] = useState<string | null>(null);
  const [renamingAreaValue, setRenamingAreaValue] = useState("");
  const [renamingProjectId, setRenamingProjectId] = useState<string | null>(null);
  const [renamingProjectValue, setRenamingProjectValue] = useState("");
  const [sidebarDrag, setSidebarDrag] = useState<SidebarReorderDrag | null>(null);
  const [sidebarInsertionTarget, setSidebarInsertionTarget] = useState<SidebarInsertionTarget | null>(null);

  const closeContextMenu = useCallback(() => setContextMenu(null), []);
  const clearSidebarDrag = useCallback(() => {
    setSidebarDrag(null);
    setSidebarInsertionTarget(null);
  }, []);

  const beginSidebarDrag = useCallback((e: React.DragEvent, drag: SidebarReorderDrag) => {
    setSidebarDrag(drag);
    setSidebarInsertionTarget(null);
    e.dataTransfer.effectAllowed = "move";
    e.dataTransfer.setData(SIDEBAR_REORDER_MIME, JSON.stringify(drag));
  }, []);

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

  const buildAreaMoves = useCallback((sourceId: string, index: number) => {
    const activeAreaIds = new Set(areas.map((area) => area.id));
    const inactiveAreas = allAreas
      .filter((area) => !activeAreaIds.has(area.id))
      .sort((a, b) => a.index - b.index);
    const sourceIndex = areas.findIndex((area) => area.id === sourceId);
    if (sourceIndex === -1) return null;

    const targetIndex = index > sourceIndex ? index - 1 : index;
    if (targetIndex === sourceIndex) return null;

    const reordered = [...areas];
    const [moved] = reordered.splice(sourceIndex, 1);
    reordered.splice(targetIndex, 0, moved);
    return [...reordered, ...inactiveAreas].map((area, nextIndex) => ({ id: area.id, index: nextIndex }));
  }, [allAreas, areas]);

  const buildProjectMoves = useCallback((areaId: string | null, sourceId: string, index: number) => {
    const activeProjectIds = new Set(projects.map((project) => project.id));
    const inactiveProjects = allProjects
      .filter((project) => !activeProjectIds.has(project.id))
      .sort((a, b) => a.index - b.index);
    const group = projects.filter((project) => project.areaId === areaId);
    const sourceIndex = group.findIndex((project) => project.id === sourceId);
    if (sourceIndex === -1) return null;

    const targetIndex = index > sourceIndex ? index - 1 : index;
    if (targetIndex === sourceIndex) return null;

    const reorderedGroup = [...group];
    const [moved] = reorderedGroup.splice(sourceIndex, 1);
    reorderedGroup.splice(targetIndex, 0, moved);

    let groupIndex = 0;
    const reorderedProjects = projects.map((project) =>
      project.areaId === areaId ? reorderedGroup[groupIndex++] : project,
    );

    return [...reorderedProjects, ...inactiveProjects].map((project, nextIndex) => ({ id: project.id, index: nextIndex }));
  }, [allProjects, projects]);

  const draggedAreaIndex = sidebarDrag?.kind === "area"
    ? areas.findIndex((area) => area.id === sidebarDrag.id)
    : -1;

  const renderAreaDropZone = (index: number) => {
    if (sidebarDrag?.kind !== "area") return null;

    const isVisible = sidebarInsertionTarget?.kind === "area"
      && sidebarInsertionTarget.index === index
      && index !== draggedAreaIndex
      && index !== draggedAreaIndex + 1;

    return (
      <div
        key={`area-drop-zone-${index}`}
        className="sidebar-drop-zone sidebar-drop-zone-area"
        onDragOver={(e) => {
          e.preventDefault();
          e.dataTransfer.dropEffect = "move";
          setSidebarInsertionTarget({ kind: "area", index });
        }}
        onDragLeave={() => {
          setSidebarInsertionTarget((current) =>
            current?.kind === "area" && current.index === index ? null : current,
          );
        }}
        onDrop={async (e) => {
          e.preventDefault();
          if (sidebarDrag?.kind !== "area") return;
          const moves = buildAreaMoves(sidebarDrag.id, index);
          clearSidebarDrag();
          if (moves) await reorderAreas(moves);
        }}
      >
        {isVisible ? <SidebarDropSlot /> : null}
      </div>
    );
  };

  const renderProjectDropZone = (areaId: string | null, index: number) => {
    if (sidebarDrag?.kind !== "project" || sidebarDrag.areaId !== areaId) return null;

    const group = projects.filter((project) => project.areaId === areaId);
    const draggedProjectIndex = group.findIndex((project) => project.id === sidebarDrag.id);
    const isVisible = sidebarInsertionTarget?.kind === "project"
      && sidebarInsertionTarget.areaId === areaId
      && sidebarInsertionTarget.index === index
      && index !== draggedProjectIndex
      && index !== draggedProjectIndex + 1;

    return (
      <div
        key={`project-drop-zone-${areaId ?? "root"}-${index}`}
        className="sidebar-drop-zone sidebar-drop-zone-project"
        onDragOver={(e) => {
          e.preventDefault();
          e.dataTransfer.dropEffect = "move";
          setSidebarInsertionTarget({ kind: "project", areaId, index });
        }}
        onDragLeave={() => {
          setSidebarInsertionTarget((current) =>
            current?.kind === "project" && current.areaId === areaId && current.index === index
              ? null
              : current,
          );
        }}
        onDrop={async (e) => {
          e.preventDefault();
          if (sidebarDrag?.kind !== "project" || sidebarDrag.areaId !== areaId) return;
          const moves = buildProjectMoves(areaId, sidebarDrag.id, index);
          clearSidebarDrag();
          if (moves) await reorderProjects(moves);
        }}
      >
        {isVisible ? <SidebarDropSlot /> : null}
      </div>
    );
  };

  return (
    <div className="sidebar">
      <div className="sidebar-toolbar" data-tauri-drag-region />

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
      {areas.map((area, areaIndex) => {
        const areaProjects = projectsByArea.get(area.id) ?? [];
        return (
          <Fragment key={area.id}>
            {renderAreaDropZone(areaIndex)}
            <div className="sidebar-group">
              {renamingAreaId === area.id ? (
                <SidebarRenameField
                  value={renamingAreaValue}
                  className="sidebar-rename-area"
                  onChange={setRenamingAreaValue}
                  onCommit={() => {
                    const title = renamingAreaValue.trim();
                    if (title) updateArea({ id: area.id, title });
                    setRenamingAreaId(null);
                  }}
                  onCancel={() => setRenamingAreaId(null)}
                />
              ) : (
                <div
                  className={`sidebar-group-label${activeView === `area-${area.id}` ? " active" : ""}`}
                  draggable
                  onClick={() => setActiveView(`area-${area.id}`)}
                  onContextMenu={(e) => handleAreaContextMenu(e, area)}
                  onDragStart={(e) => beginSidebarDrag(e, { kind: "area", id: area.id })}
                  onDragEnd={clearSidebarDrag}
                >
                  <span className="sidebar-group-label-text">{area.title}</span>
                  <Button
                    className="sidebar-add-btn"
                    variant="ghost"
                    size="sm"
                    title={`Add project to ${area.title}`}
                    onClick={async (e) => {
                      e.stopPropagation();
                      const project = await createProject({ title: "New Project", areaId: area.id });
                      if (project) {
                        setRenamingProjectId(project.id);
                        setRenamingProjectValue("New Project");
                      }
                    }}
                  >
                    +
                  </Button>
                </div>
              )}
              {areaProjects.map((p, projectIndex) => (
                <Fragment key={p.id}>
                  {renderProjectDropZone(area.id, projectIndex)}
                  <ProjectItem
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
                    onTaskDrop={(taskId, projectId) => updateTask({ id: taskId, projectId })}
                    draggable={renamingProjectId !== p.id}
                    onDragStart={(e) => beginSidebarDrag(e, {
                      kind: "project",
                      id: p.id,
                      areaId: area.id,
                    })}
                    onDragEnd={clearSidebarDrag}
                  />
                </Fragment>
              ))}
              {renderProjectDropZone(area.id, areaProjects.length)}
            </div>
          </Fragment>
        );
      })}
      {renderAreaDropZone(areas.length)}

      {/* Standalone projects (no area) — same visual style, no heading */}
      {(() => {
        const standalone = projectsByArea.get(null) ?? [];
        if (standalone.length === 0) return null;
        return (
          <div className="sidebar-group">
            {standalone.map((p, index) => (
              <Fragment key={p.id}>
                {renderProjectDropZone(null, index)}
                <ProjectItem
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
                  onTaskDrop={(taskId, projectId) => updateTask({ id: taskId, projectId })}
                  draggable={renamingProjectId !== p.id}
                  onDragStart={(e) => beginSidebarDrag(e, {
                    kind: "project",
                    id: p.id,
                    areaId: null,
                  })}
                  onDragEnd={clearSidebarDrag}
                />
              </Fragment>
            ))}
            {renderProjectDropZone(null, standalone.length)}
          </div>
        );
      })()}

      {/* + New Area — creates area then enters rename mode */}
      <div className="sidebar-footer-block">
        <SidebarRow
          className="sidebar-item-muted"
          onClick={async () => {
            const area = await createArea({ title: "New Area" });
            if (area) {
              setRenamingAreaId(area.id);
              setRenamingAreaValue("New Area");
            }
          }}
        >
          <span className="sidebar-icon" style={{ fontSize: "var(--text-base)" }}>+</span>
          <span>New Area</span>
        </SidebarRow>
      </div>

      {/* Settings — pinned to bottom */}
      <div className="sidebar-settings">
        <SidebarRow
          active={activeView === 'settings'}
          onClick={() => setActiveView('settings')}
        >
          <span className="sidebar-icon">
            <svg viewBox="0 0 16 16" style={{ width: 16, height: 16 }}>
              <circle cx="8" cy="8" r="2.5" fill="none" stroke="currentColor" strokeWidth="1.5" />
              <path d="M8 1.5v2M8 12.5v2M1.5 8h2M12.5 8h2M3.2 3.2l1.4 1.4M11.4 11.4l1.4 1.4M3.2 12.8l1.4-1.4M11.4 4.6l1.4-1.4" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
            </svg>
          </span>
          <span>Settings</span>
        </SidebarRow>
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
