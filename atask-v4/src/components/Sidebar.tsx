import { Fragment, useCallback, useEffect, useRef, useState } from "react";
import { useStore } from "@nanostores/react";
import {
  $activeView,
  $areas,
  $projects,
  $tasks,
  $taskPointerDrag,
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
import type { ActiveView, Area, Project } from "../types";
import { todayLocal, tomorrowLocal } from "../lib/dates";
import ContextMenu, { type MenuItem } from "./ContextMenu";
import { Button } from "../ui";
import { LogbookIcon, InboxIcon, SomedayIcon, TodayIcon, UpcomingIcon } from "./sidebar/SidebarIcons";
import useForeignDropIndex from '../hooks/useForeignDropIndex';
import {
  NavItem,
  ProjectItem,
  SidebarDropSlot,
  SidebarRenameField,
  SidebarRow,
  shouldHandleSidebarRowPointerDown,
} from "./sidebar/SidebarParts";
import DragOverlay from "./DragOverlay";
import usePointerReorder from "../hooks/usePointerReorder";

type ContextMenuState =
  | { kind: "project"; project: Project; position: { x: number; y: number } }
  | { kind: "area"; area: Area; position: { x: number; y: number } }
  | null;

type SidebarProjectGroupProps = {
  areaId: string | null;
  projects: Project[];
  activeView: ActiveView;
  projectTaskCounts: Map<string, number>;
  renamingProjectId: string | null;
  renamingProjectValue: string;
  onRenamingValueChange: (value: string) => void;
  onRenameCommit: () => void;
  onRenameCancel: () => void;
  onTaskDrop: (taskId: string, projectId: string) => void;
  onProjectContextMenu: (e: React.MouseEvent, project: Project) => void;
  onProjectReorder: (areaId: string | null, orderedIds: string[]) => Promise<void>;
  onProjectCrossAreaDrop: (projectId: string, targetAreaId: string | null) => void;
  setActiveView: (view: ActiveView) => void;
};

function SidebarProjectGroup({
  areaId,
  projects,
  activeView,
  projectTaskCounts,
  renamingProjectId,
  renamingProjectValue,
  onRenamingValueChange,
  onRenameCommit,
  onRenameCancel,
  onTaskDrop,
  onProjectContextMenu,
  onProjectReorder,
  onProjectCrossAreaDrop,
  setActiveView,
}: SidebarProjectGroupProps) {
  // Each project group is its own list — use the area id (or "root") so
  // foreign drag detection can tell groups apart and exclude the source.
  const projectListId = `projects:${areaId ?? 'root'}`;
  const projectListRef = useRef<HTMLDivElement | null>(null);
  const { reorderState, getPointerHandlers, registerItem, getItemRect } = usePointerReorder({
    items: projects,
    listId: projectListId,
    kind: 'project',
    onReorder: async (moves) => {
      const orderedIds = [...moves].sort((left, right) => left.index - right.index).map((move) => move.id);
      await onProjectReorder(areaId, orderedIds);
    },
    shouldHandlePointerDown: (event) => shouldHandleSidebarRowPointerDown(event.target),
    // Intentionally NOT wiring onDragStart to startTaskPointerDrag here:
    // that atom represents a TASK drag and other sidebar items (project
    // rows, area labels) branch on `taskDrag.activeTaskId !== null` to
    // render themselves as task drop targets. Setting it during a
    // project drag made projects in other areas light up with the
    // "task dragging over me" style, which confused users. The drop
    // still works via onCrossListDrop below; visual feedback for
    // project drags is provided by DragOverlay's floating clone.
    onCrossListDrop: (projectId, target) => {
      // Project-to-area drag: `target` is the closest ancestor with
      // data-sidebar-item-id that the pointer released over. Because
      // project rows and area labels are SIBLINGS (both live under
      // .sidebar-group), we can't use .closest("area") — projects have
      // no area ancestor in the DOM tree. Check both cases explicitly:
      //   - drop on area label -> move to that area
      //   - drop on a project in another area -> move to that project's area
      const kind = target.getAttribute('data-sidebar-item-kind');
      if (kind === 'area') {
        const targetAreaId = target.getAttribute('data-sidebar-item-id');
        if (targetAreaId && targetAreaId !== areaId) {
          onProjectCrossAreaDrop(projectId, targetAreaId);
          return true;
        }
      }
      if (kind === 'project') {
        const targetProjectId = target.getAttribute('data-sidebar-item-id');
        if (targetProjectId && targetProjectId !== projectId) {
          const targetProject = $projects.get().find((p) => p.id === targetProjectId);
          if (targetProject) {
            const targetAreaId = targetProject.areaId ?? null;
            if (targetAreaId !== areaId) {
              onProjectCrossAreaDrop(projectId, targetAreaId);
              return true;
            }
          }
        }
      }
      return false;
    },
  });

  // Parallel ref map so the foreign-drop computation can read item rects
  // without poking into usePointerReorder's private itemElementsRef.
  // MUST be declared BEFORE useForeignDropIndex — that hook calls
  // getItemElements() during render when a foreign drag is active, and
  // the closure reads `projectItemRefs`. JavaScript const/let TDZ rules
  // mean the closure throws if the ref is declared further down the
  // function body.
  const projectItemRefs = useRef<Map<string, HTMLDivElement>>(new Map());

  // Foreign drop indicator: when a project drag from ANOTHER area's group
  // is hovering this group, compute where it would land in our list.
  const foreignDrop = useForeignDropIndex({
    listId: projectListId,
    kind: 'project',
    containerRef: projectListRef,
    getItemElements: () => {
      const elements: HTMLElement[] = [];
      for (const project of projects) {
        const el = projectItemRefs.current.get(project.id);
        if (el) elements.push(el);
      }
      return elements;
    },
  });

  const draggedProjectIndex = reorderState.activeId
    ? projects.findIndex((project) => project.id === reorderState.activeId)
    : -1;

  const itemWidth = reorderState.activeId ? getItemRect(reorderState.activeId)?.width ?? null : null;

  // Cache the per-id ref callback so React sees a stable function across
  // re-renders. Without this, every render of SidebarProjectGroup (which
  // happens on every cursor move during a drag because $pointerDragCursor
  // updates trigger useStore subscribers) creates a fresh closure for
  // every project, churning the ref maps and breaking pointer-event
  // continuity. The cache invalidates whenever registerItem changes
  // (which is stable in practice — usePointerReorder memoizes it with []
  // deps).
  const registerProjectItemCacheRef = useRef<Map<string, (node: HTMLDivElement | null) => void>>(
    new Map(),
  );
  useEffect(() => {
    // Drop the cache whenever the underlying register function rotates.
    registerProjectItemCacheRef.current = new Map();
  }, [registerItem]);
  const registerProjectItem = useCallback((id: string) => {
    let cached = registerProjectItemCacheRef.current.get(id);
    if (!cached) {
      const hookRegister = registerItem(id);
      cached = (node: HTMLDivElement | null) => {
        if (node) projectItemRefs.current.set(id, node);
        else projectItemRefs.current.delete(id);
        hookRegister(node);
      };
      registerProjectItemCacheRef.current.set(id, cached);
    }
    return cached;
  }, [registerItem]);

  const renderDragClone = (id: string) => {
    const project = projects.find((p) => p.id === id);
    if (!project) return null;
    return (
      <div
        style={{
          background: 'var(--sidebar-hover)',
          borderRadius: 'var(--radius-md)',
          boxShadow: '0 4px 12px rgba(0,0,0,0.15)',
          padding: '8px 12px',
        }}
      >
        <span style={{ fontSize: 'var(--text-sm)', color: 'var(--ink-primary)' }}>
          {project.title}
        </span>
      </div>
    );
  };

  const renderDropZone = (index: number) => {
    // Native within-list drop indicator.
    const localVisible =
      reorderState.isPointerDragging &&
      reorderState.dropIndex === index &&
      index !== draggedProjectIndex &&
      index !== draggedProjectIndex + 1;

    // Foreign drop indicator: another area's project is being dragged
    // over our list. We don't have a native drag of our own, so just
    // mirror the foreign dropIndex into a slot at this position.
    const foreignVisible =
      !reorderState.isPointerDragging &&
      foreignDrop.isForeignHovering &&
      foreignDrop.dropIndex === index;

    if (!localVisible && !foreignVisible) return null;

    return (
      <div
        key={`project-drop-zone-${areaId ?? "root"}-${index}`}
        className="sidebar-drop-zone sidebar-drop-zone-project"
      >
        <SidebarDropSlot />
      </div>
    );
  };

  return (
    <div ref={projectListRef} className="sidebar-project-list">
      {projects.map((project, projectIndex) => (
        <Fragment key={project.id}>
          {renderDropZone(projectIndex)}
          <ProjectItem
            project={project}
            badge={projectTaskCounts.get(project.id) ?? 0}
            activeView={activeView}
            onClick={setActiveView}
            onContextMenu={onProjectContextMenu}
            isRenaming={renamingProjectId === project.id}
            renamingValue={renamingProjectValue}
            onRenamingValueChange={onRenamingValueChange}
            onRenameCommit={onRenameCommit}
            onRenameCancel={onRenameCancel}
            onTaskDrop={onTaskDrop}
            reorderRef={registerProjectItem(project.id)}
            reorderHandlers={getPointerHandlers(project.id)}
            isReordering={reorderState.activeId === project.id}
          />
        </Fragment>
      ))}
      {renderDropZone(projects.length)}
      <DragOverlay
        activeId={reorderState.activeId}
        cursorX={reorderState.cursorX}
        cursorY={reorderState.cursorY}
        itemWidth={itemWidth}
        renderClone={renderDragClone}
      />
    </div>
  );
}

export default function Sidebar() {
  const activeView = useStore($activeView);
  const allProjects = useStore($projects);
  const allAreas = useStore($areas);
  const tasks = useStore($tasks);
  const setActiveView = (view: ActiveView) => $activeView.set(view);

  const projects = [...allProjects].filter((project) => project.status === 0).sort((a, b) => a.index - b.index);
  const areas = [...allAreas].filter((area) => !area.archived).sort((a, b) => a.index - b.index);

  const [contextMenu, setContextMenu] = useState<ContextMenuState>(null);
  const [renamingAreaId, setRenamingAreaId] = useState<string | null>(null);
  const [renamingAreaValue, setRenamingAreaValue] = useState("");
  const [hoveredAreaId, setHoveredAreaId] = useState<string | null>(null);
  const taskDrag = useStore($taskPointerDrag);
  const isTaskDragActive = taskDrag.activeTaskId !== null;
  const [renamingProjectId, setRenamingProjectId] = useState<string | null>(null);
  const [renamingProjectValue, setRenamingProjectValue] = useState("");

  const closeContextMenu = useCallback(() => setContextMenu(null), []);

  const handleProjectContextMenu = useCallback(
    (event: React.MouseEvent, project: Project) => {
      event.preventDefault();
      event.stopPropagation();
      setContextMenu({ kind: "project", project, position: { x: event.clientX, y: event.clientY } });
    },
    [],
  );

  const handleAreaContextMenu = useCallback(
    (event: React.MouseEvent, area: Area) => {
      event.preventDefault();
      event.stopPropagation();
      setContextMenu({ kind: "area", area, position: { x: event.clientX, y: event.clientY } });
    },
    [],
  );

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

      items.push({
        label: "No Area",
        onClick: () => moveProjectToArea(project.id, null),
        disabled: project.areaId == null,
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
    if (!renamingProjectId) return;

    const title = renamingProjectValue.trim();
    if (title) {
      updateProject({ id: renamingProjectId, title });
    }

    setRenamingProjectId(null);
  }, [renamingProjectId, renamingProjectValue]);

  const cancelProjectRename = useCallback(() => {
    setRenamingProjectId(null);
  }, []);

  const inboxCount = tasks.filter((task) => task.schedule === 0 && task.status === 0 && task.projectId === null).length;
  const todayCount = tasks.filter((task) => task.schedule === 1 && task.status === 0).length;

  const projectTaskCounts = new Map<string, number>();
  for (const task of tasks) {
    if (task.projectId && task.status === 0) {
      projectTaskCounts.set(task.projectId, (projectTaskCounts.get(task.projectId) ?? 0) + 1);
    }
  }

  const projectsByArea = new Map<string | null, Project[]>();
  for (const project of projects) {
    const areaKey = project.areaId ?? null;
    const list = projectsByArea.get(areaKey) ?? [];
    list.push(project);
    projectsByArea.set(areaKey, list);
  }

  const contextMenuItems: MenuItem[] =
    contextMenu?.kind === "project"
      ? buildProjectMenuItems(contextMenu.project)
      : contextMenu?.kind === "area"
        ? buildAreaMenuItems(contextMenu.area)
        : [];

  const buildAreaMoves = useCallback((orderedIds: string[]) => {
    const inactiveAreas = allAreas
      .filter((area) => area.archived)
      .sort((a, b) => a.index - b.index);
    const reorderedAreas = orderedIds
      .map((id) => areas.find((area) => area.id === id) ?? null)
      .filter((area): area is Area => area !== null);

    if (reorderedAreas.length !== areas.length) return null;

    const orderChanged = reorderedAreas.some((area, index) => area.id !== areas[index]?.id);
    if (!orderChanged) return null;

    return [...reorderedAreas, ...inactiveAreas].map((area, index) => ({ id: area.id, index }));
  }, [allAreas, areas]);

  const buildProjectMoves = useCallback((areaId: string | null, orderedIds: string[]) => {
    const activeProjectIds = new Set(projects.map((project) => project.id));
    const inactiveProjects = allProjects
      .filter((project) => !activeProjectIds.has(project.id))
      .sort((a, b) => a.index - b.index);
    const group = projects.filter((project) => (project.areaId ?? null) === areaId);
    const reorderedGroup = orderedIds
      .map((id) => group.find((project) => project.id === id) ?? null)
      .filter((project): project is Project => project !== null);

    if (reorderedGroup.length !== group.length) return null;

    const orderChanged = reorderedGroup.some((project, index) => project.id !== group[index]?.id);
    if (!orderChanged) return null;

    let groupIndex = 0;
    const reorderedProjects = projects.map((project) =>
      (project.areaId ?? null) === areaId ? reorderedGroup[groupIndex++] : project,
    );

    return [...reorderedProjects, ...inactiveProjects].map((project, index) => ({ id: project.id, index }));
  }, [allProjects, projects]);

  const areaReorder = usePointerReorder({
    items: areas,
    onReorder: async (moves) => {
      const orderedIds = [...moves].sort((left, right) => left.index - right.index).map((move) => move.id);
      const nextMoves = buildAreaMoves(orderedIds);
      if (nextMoves) {
        await reorderAreas(nextMoves);
      }
    },
    shouldHandlePointerDown: (event) => shouldHandleSidebarRowPointerDown(event.target),
  });

  const draggedAreaIndex = areaReorder.reorderState.activeId
    ? areas.findIndex((area) => area.id === areaReorder.reorderState.activeId)
    : -1;

  const renderAreaDropZone = (index: number) => {
    if (!areaReorder.reorderState.isPointerDragging) return null;

    const isVisible = areaReorder.reorderState.dropIndex === index
      && index !== draggedAreaIndex
      && index !== draggedAreaIndex + 1;

    if (!isVisible) return null;

    return (
      <div
        key={`area-drop-zone-${index}`}
        className="sidebar-drop-zone sidebar-drop-zone-area"
      >
        <SidebarDropSlot />
      </div>
    );
  };

  const areaItemWidth = areaReorder.reorderState.activeId ? areaReorder.getItemRect(areaReorder.reorderState.activeId)?.width ?? null : null;

  const renderAreaDragClone = (id: string) => {
    const area = areas.find((a) => a.id === id);
    if (!area) return null;
    return (
      <div
        style={{
          background: 'var(--sidebar-hover)',
          borderRadius: 'var(--radius-md)',
          boxShadow: '0 4px 12px rgba(0,0,0,0.15)',
          padding: '8px 12px',
        }}
      >
        <span style={{ fontSize: 'var(--text-sm)', color: 'var(--ink-primary)' }}>
          {area.title}
        </span>
      </div>
    );
  };

  return (
    <nav className="sidebar" aria-label="Main navigation">
      <div className="sidebar-toolbar" data-tauri-drag-region />

      <div className="sidebar-group">
        <NavItem
          view="inbox"
          label="Inbox"
          icon={<InboxIcon />}
          badge={inboxCount}
          activeView={activeView}
          onClick={setActiveView}
          onTaskDrop={(taskId) => updateTask({ id: taskId, schedule: 0, startDate: null, timeSlot: null, projectId: null, areaId: null, sectionId: null })}
        />
        <NavItem
          view="today"
          label="Today"
          icon={<TodayIcon />}
          badge={todayCount}
          activeView={activeView}
          onClick={setActiveView}
          onTaskDrop={(taskId) => {
            const today = todayLocal();
            updateTask({ id: taskId, schedule: 1, startDate: today, projectId: null, areaId: null, sectionId: null });
          }}
        />
        <NavItem
          view="upcoming"
          label="Upcoming"
          icon={<UpcomingIcon />}
          activeView={activeView}
          onClick={setActiveView}
          onTaskDrop={(taskId) => {
            updateTask({ id: taskId, schedule: 3, startDate: tomorrowLocal(), projectId: null, areaId: null, sectionId: null });
          }}
        />
        <NavItem
          view="someday"
          label="Someday"
          icon={<SomedayIcon />}
          activeView={activeView}
          onClick={setActiveView}
          onTaskDrop={(taskId) => updateTask({ id: taskId, schedule: 2, projectId: null, areaId: null, sectionId: null })}
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

      {areas.map((area, areaIndex) => {
        const areaProjects = projectsByArea.get(area.id) ?? [];
        const areaReorderHandlers = areaReorder.getPointerHandlers(area.id);
        const isAreaReordering = areaReorder.reorderState.activeId === area.id;
        const isAreaDropTarget = (hoveredAreaId === area.id || taskDrag.hoverTargetId === area.id) && isTaskDragActive;

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
                    if (title) {
                      updateArea({ id: area.id, title });
                    }
                    setRenamingAreaId(null);
                  }}
                  onCancel={() => setRenamingAreaId(null)}
                />
              ) : (
                <div
                  ref={areaReorder.registerItem(area.id)}
                  className={`sidebar-group-label${activeView === `area-${area.id}` ? " active" : ""}${isAreaReordering ? " sidebar-item-dragging" : ""}${isAreaDropTarget ? " drag-target" : ""}`}
                  data-sidebar-item-id={area.id}
                  data-sidebar-item-kind="area"
                  role="button"
                  tabIndex={0}
                  aria-current={activeView === `area-${area.id}` ? "page" : undefined}
                  aria-label={`Area: ${area.title}`}
                  onKeyDown={(e) => {
                    if (e.key === "Enter" || e.key === " ") {
                      e.preventDefault();
                      setActiveView(`area-${area.id}`);
                    }
                  }}
                  onClick={() => setActiveView(`area-${area.id}`)}
                  onContextMenu={(event) => handleAreaContextMenu(event, area)}
                  onPointerDown={areaReorderHandlers.onPointerDown}
                  onMouseDown={areaReorderHandlers.onMouseDown}
                  onPointerEnter={() => isTaskDragActive && setHoveredAreaId(area.id)}
                  onPointerLeave={() => setHoveredAreaId(null)}
                >
                  <span className="sidebar-group-label-text">{area.title}</span>
                  <Button
                    className="sidebar-add-btn"
                    variant="ghost"
                    size="sm"
                    data-reorder-ignore
                    title={`Add project to ${area.title}`}
                    onClick={async (event) => {
                      event.stopPropagation();
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

              <SidebarProjectGroup
                areaId={area.id}
                projects={areaProjects}
                activeView={activeView}
                projectTaskCounts={projectTaskCounts}
                renamingProjectId={renamingProjectId}
                renamingProjectValue={renamingProjectValue}
                onRenamingValueChange={setRenamingProjectValue}
                onRenameCommit={commitProjectRename}
                onRenameCancel={cancelProjectRename}
                onTaskDrop={(taskId, projectId) => updateTask({ id: taskId, projectId })}
                onProjectContextMenu={handleProjectContextMenu}
                onProjectCrossAreaDrop={(projectId, targetAreaId) => moveProjectToArea(projectId, targetAreaId)}
                onProjectReorder={async (nextAreaId, orderedIds) => {
                  const nextMoves = buildProjectMoves(nextAreaId, orderedIds);
                  if (nextMoves) {
                    await reorderProjects(nextMoves);
                  }
                }}
                setActiveView={setActiveView}
              />
            </div>
          </Fragment>
        );
      })}
      {renderAreaDropZone(areas.length)}

      {(() => {
        const standaloneProjects = projectsByArea.get(null) ?? [];
        if (standaloneProjects.length === 0) return null;

        return (
          <div className="sidebar-group">
            <SidebarProjectGroup
              areaId={null}
              projects={standaloneProjects}
              activeView={activeView}
              projectTaskCounts={projectTaskCounts}
              renamingProjectId={renamingProjectId}
              renamingProjectValue={renamingProjectValue}
              onRenamingValueChange={setRenamingProjectValue}
              onRenameCommit={commitProjectRename}
              onRenameCancel={cancelProjectRename}
              onTaskDrop={(taskId, projectId) => updateTask({ id: taskId, projectId })}
              onProjectContextMenu={handleProjectContextMenu}
              onProjectCrossAreaDrop={(projectId, targetAreaId) => moveProjectToArea(projectId, targetAreaId)}
              onProjectReorder={async (nextAreaId, orderedIds) => {
                const nextMoves = buildProjectMoves(nextAreaId, orderedIds);
                if (nextMoves) {
                  await reorderProjects(nextMoves);
                }
              }}
              setActiveView={setActiveView}
            />
          </div>
        );
      })()}

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

      <div className="sidebar-settings">
        <SidebarRow
          active={activeView === "settings"}
          onClick={() => setActiveView("settings")}
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

      {contextMenu && (
        <ContextMenu
          items={contextMenuItems}
          position={contextMenu.position}
          onClose={closeContextMenu}
        />
      )}

      <DragOverlay
        activeId={areaReorder.reorderState.activeId}
        cursorX={areaReorder.reorderState.cursorX}
        cursorY={areaReorder.reorderState.cursorY}
        itemWidth={areaItemWidth}
        renderClone={renderAreaDragClone}
      />
    </nav>
  );
}
