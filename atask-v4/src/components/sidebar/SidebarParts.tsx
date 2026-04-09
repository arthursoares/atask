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
  ariaLabel,
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
  ariaLabel?: string;
  onClick?: () => void;
  onContextMenu?: (e: React.MouseEvent) => void;
  onDragOver?: (e: React.DragEvent) => void;
  onDragLeave?: () => void;
  onDrop?: (e: React.DragEvent) => void;
  onPointerEnter?: () => void;
  onPointerLeave?: () => void;
}) {
  // Handle keyboard activation: Enter or Space triggers onClick so the row
  // is reachable via Tab and usable by keyboard-only + screen-reader users.
  // We keep the element as a <div role="button"> rather than a real <button>
  // to avoid fighting the pointer-reorder handlers (button elements swallow
  // pointer events for their own click synthesis) and to preserve the
  // flexible icon + label + badge child layout.
  const handleKeyDown = (e: React.KeyboardEvent<HTMLDivElement>) => {
    if (!onClick) return;
    if (e.key === "Enter" || e.key === " ") {
      e.preventDefault();
      onClick();
    }
  };
  return (
    <div
      ref={reorderRef}
      className={`sidebar-item${active ? " active" : ""}${isDragTarget ? " drag-target" : ""}${isReordering ? " sidebar-item-dragging" : ""}${className ? ` ${className}` : ""}`}
      data-sidebar-item-id={dataSidebarItemId}
      data-sidebar-item-kind={dataSidebarItemKind}
      role={onClick ? "button" : undefined}
      tabIndex={onClick ? 0 : undefined}
      aria-current={active ? "page" : undefined}
      aria-label={ariaLabel}
      onKeyDown={onClick ? handleKeyDown : undefined}
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
  const [nativeDragOver, setNativeDragOver] = useState(false);
  const taskDrag = useStore($taskPointerDrag);
  const isDragTarget = nativeDragOver || (taskDrag.activeTaskId !== null && taskDrag.hoverTargetId === view);

  return (
    <SidebarRow
      active={activeView === view}
      isDragTarget={isDragTarget}
      dataSidebarItemId={view}
      dataSidebarItemKind="nav"
      onClick={() => onClick(view)}
      onDragOver={onTaskDrop ? (e) => {
        if (!isTaskTransfer(e)) return;
        e.preventDefault();
        e.dataTransfer.dropEffect = "move";
        setNativeDragOver(true);
      } : undefined}
      onDragLeave={onTaskDrop ? () => setNativeDragOver(false) : undefined}
      onDrop={onTaskDrop ? (e) => {
        if (!isTaskTransfer(e)) return;
        e.preventDefault();
        setNativeDragOver(false);
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
  const [nativeDragOver, setNativeDragOver] = useState(false);
  const taskDrag = useStore($taskPointerDrag);
  const isDragTarget = nativeDragOver || (taskDrag.activeTaskId !== null && taskDrag.hoverTargetId === project.id);

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
      onDragOver={(e) => {
        if (!isTaskTransfer(e)) return;
        e.preventDefault();
        e.dataTransfer.dropEffect = "move";
        setNativeDragOver(true);
      }}
      onDragLeave={() => setNativeDragOver(false)}
      onDrop={(e) => {
        if (!isTaskTransfer(e)) return;
        e.preventDefault();
        setNativeDragOver(false);
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
