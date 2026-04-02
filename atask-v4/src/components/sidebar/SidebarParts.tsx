import { useState } from "react";
import { useStore } from "@nanostores/react";
import type { ActiveView, Project } from "../../types";
import { Field } from "../../ui";
import { $taskPointerDrag } from "../../store/ui";

function hasDragType(e: React.DragEvent, type: string) {
  return Array.from(e.dataTransfer.types).includes(type);
}

function isTaskTransfer(e: React.DragEvent) {
  return hasDragType(e, "text/plain");
}

export function shouldHandleSidebarRowPointerDown(target: EventTarget | null): boolean {
  if (!(target instanceof Element)) {
    return true;
  }

  return target.closest("[data-reorder-ignore]") === null;
}

export function SidebarRow({
  active = false,
  children,
  className = "",
  isDragTarget = false,
  dataSidebarItemId,
  dataSidebarItemKind,
  reorderRef,
  reorderHandlers,
  isReordering = false,
  onClick,
  onContextMenu,
  onDragOver,
  onDragLeave,
  onDrop,
  onPointerEnter,
  onPointerLeave,
}: {
  active?: boolean;
  children: React.ReactNode;
  className?: string;
  isDragTarget?: boolean;
  dataSidebarItemId?: string;
  dataSidebarItemKind?: string;
  reorderRef?: (node: HTMLDivElement | null) => void;
  reorderHandlers?: {
    onPointerDown: (e: React.PointerEvent<HTMLDivElement>) => void;
    onMouseDown: (e: React.MouseEvent<HTMLDivElement>) => void;
  };
  isReordering?: boolean;
  onClick?: () => void;
  onContextMenu?: (e: React.MouseEvent) => void;
  onDragOver?: (e: React.DragEvent) => void;
  onDragLeave?: () => void;
  onDrop?: (e: React.DragEvent) => void;
  onPointerEnter?: () => void;
  onPointerLeave?: () => void;
}) {
  return (
    <div
      ref={reorderRef}
      className={`sidebar-item${active ? " active" : ""}${isDragTarget ? " drag-target" : ""}${isReordering ? " sidebar-item-dragging" : ""}${className ? ` ${className}` : ""}`}
      data-sidebar-item-id={dataSidebarItemId}
      data-sidebar-item-kind={dataSidebarItemKind}
      onClick={onClick}
      onContextMenu={onContextMenu}
      onPointerDown={reorderHandlers?.onPointerDown}
      onMouseDown={reorderHandlers?.onMouseDown}
      onPointerEnter={onPointerEnter}
      onPointerLeave={onPointerLeave}
      onDragOver={onDragOver}
      onDragLeave={onDragLeave}
      onDrop={onDrop}
    >
      {children}
    </div>
  );
}

export function SidebarDropSlot() {
  return (
    <div className="sidebar-drop-slot" aria-hidden="true">
      <span className="sidebar-drop-slot-dot" />
    </div>
  );
}

export function SidebarRenameField({
  value,
  onChange,
  onCommit,
  onCancel,
  placeholder,
  className = "",
}: {
  value: string;
  onChange: (value: string) => void;
  onCommit: () => void;
  onCancel: () => void;
  placeholder?: string;
  className?: string;
}) {
  return (
    <div className={`sidebar-rename-wrap${className ? ` ${className}` : ""}`}>
      <Field
        autoFocus
        value={value}
        placeholder={placeholder}
        className="sidebar-rename-input"
        onChange={(e) => onChange(e.target.value)}
        onKeyDown={(e) => {
          if (e.key === "Enter") {
            onCommit();
          } else if (e.key === "Escape") {
            onCancel();
          }
        }}
        onBlur={onCommit}
        onClick={(e) => e.stopPropagation()}
      />
    </div>
  );
}

export function NavItem({
  view,
  label,
  icon,
  badge,
  activeView,
  onClick,
  onTaskDrop,
}: {
  view: ActiveView;
  label: string;
  icon: React.ReactNode;
  badge?: number;
  activeView: ActiveView;
  onClick: (view: ActiveView) => void;
  onTaskDrop?: (taskId: string) => void;
}) {
  const [isDragTarget, setIsDragTarget] = useState(false);
  const taskDrag = useStore($taskPointerDrag);

  const handlePointerEnter = () => {
    if (taskDrag.activeTaskId) {
      setIsDragTarget(true);
    }
  };

  const handlePointerLeave = () => {
    setIsDragTarget(false);
  };

  return (
    <SidebarRow
      active={activeView === view}
      isDragTarget={isDragTarget}
      dataSidebarItemId={view}
      dataSidebarItemKind="nav"
      onClick={() => onClick(view)}
      onPointerEnter={handlePointerEnter}
      onPointerLeave={handlePointerLeave}
      onDragOver={onTaskDrop ? (e) => {
        if (!isTaskTransfer(e)) return;
        e.preventDefault();
        e.dataTransfer.dropEffect = "move";
        setIsDragTarget(true);
      } : undefined}
      onDragLeave={onTaskDrop ? () => setIsDragTarget(false) : undefined}
      onDrop={onTaskDrop ? (e) => {
        if (!isTaskTransfer(e)) return;
        e.preventDefault();
        setIsDragTarget(false);
        const taskId = e.dataTransfer.getData("text/plain");
        if (taskId) onTaskDrop(taskId);
      } : undefined}
    >
      <span className="sidebar-icon">{icon}</span>
      <span>{label}</span>
      {badge != null && badge > 0 && <span className="sidebar-badge">{badge}</span>}
    </SidebarRow>
  );
}

export function ProjectItem({
  project,
  badge,
  activeView,
  onClick,
  onContextMenu,
  isRenaming,
  renamingValue,
  onRenamingValueChange,
  onRenameCommit,
  onRenameCancel,
  onTaskDrop,
  reorderRef,
  reorderHandlers,
  isReordering = false,
}: {
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
  onTaskDrop: (taskId: string, projectId: string) => void;
  reorderRef?: (node: HTMLDivElement | null) => void;
  reorderHandlers?: {
    onPointerDown: (e: React.PointerEvent<HTMLDivElement>) => void;
    onMouseDown: (e: React.MouseEvent<HTMLDivElement>) => void;
  };
  isReordering?: boolean;
}) {
  const view: ActiveView = `project-${project.id}`;
  const [isDragTarget, setIsDragTarget] = useState(false);
  const taskDrag = useStore($taskPointerDrag);

  const handlePointerEnter = () => {
    if (taskDrag.activeTaskId) {
      setIsDragTarget(true);
    }
  };

  const handlePointerLeave = () => {
    setIsDragTarget(false);
  };

  return (
    <SidebarRow
      active={activeView === view}
      className="sidebar-item-project"
      dataSidebarItemId={project.id}
      dataSidebarItemKind="project"
      isDragTarget={isDragTarget}
      reorderRef={reorderRef}
      reorderHandlers={reorderHandlers}
      isReordering={isReordering}
      onClick={() => onClick(view)}
      onContextMenu={(e) => onContextMenu(e, project)}
      onPointerEnter={handlePointerEnter}
      onPointerLeave={handlePointerLeave}
      onDragOver={(e) => {
        if (!isTaskTransfer(e)) return;
        e.preventDefault();
        e.dataTransfer.dropEffect = "move";
        setIsDragTarget(true);
      }}
      onDragLeave={() => setIsDragTarget(false)}
      onDrop={(e) => {
        if (!isTaskTransfer(e)) return;
        e.preventDefault();
        setIsDragTarget(false);
        const taskId = e.dataTransfer.getData("text/plain");
        if (taskId) onTaskDrop(taskId, project.id);
      }}
    >
      <span
        className="sidebar-dot"
        style={{ background: project.color || "var(--accent)" }}
      />
      {isRenaming ? (
        <Field
          autoFocus
          value={renamingValue}
          className="sidebar-rename-input sidebar-rename-input-project"
          data-reorder-ignore
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
        />
      ) : (
        <span>{project.title}</span>
      )}
      {badge > 0 && !isRenaming && <span className="sidebar-badge">{badge}</span>}
    </SidebarRow>
  );
}
