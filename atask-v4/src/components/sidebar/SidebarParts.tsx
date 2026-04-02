import { useState } from "react";
import type { ActiveView, Project } from "../../types";
import { Field } from "../../ui";

export const SIDEBAR_REORDER_MIME = "application/x-atask-sidebar-item";

function hasDragType(e: React.DragEvent, type: string) {
  return Array.from(e.dataTransfer.types).includes(type);
}

function isTaskTransfer(e: React.DragEvent) {
  return hasDragType(e, "text/plain") && !hasDragType(e, SIDEBAR_REORDER_MIME);
}

export function SidebarRow({
  active = false,
  children,
  className = "",
  isDragTarget = false,
  draggable,
  onClick,
  onContextMenu,
  onDragStart,
  onDragEnd,
  onDragOver,
  onDragLeave,
  onDrop,
}: {
  active?: boolean;
  children: React.ReactNode;
  className?: string;
  isDragTarget?: boolean;
  draggable?: boolean;
  onClick?: () => void;
  onContextMenu?: (e: React.MouseEvent) => void;
  onDragStart?: (e: React.DragEvent) => void;
  onDragEnd?: () => void;
  onDragOver?: (e: React.DragEvent) => void;
  onDragLeave?: () => void;
  onDrop?: (e: React.DragEvent) => void;
}) {
  return (
    <div
      className={`sidebar-item${active ? " active" : ""}${isDragTarget ? " drag-target" : ""}${className ? ` ${className}` : ""}`}
      draggable={draggable}
      onClick={onClick}
      onContextMenu={onContextMenu}
      onDragStart={onDragStart}
      onDragEnd={onDragEnd}
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

  return (
    <SidebarRow
      active={activeView === view}
      isDragTarget={isDragTarget}
      onClick={() => onClick(view)}
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
  draggable,
  onDragStart,
  onDragEnd,
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
  draggable?: boolean;
  onDragStart?: (e: React.DragEvent) => void;
  onDragEnd?: () => void;
}) {
  const view: ActiveView = `project-${project.id}`;
  const [isDragTarget, setIsDragTarget] = useState(false);

  return (
    <SidebarRow
      active={activeView === view}
      className="sidebar-item-project"
      isDragTarget={isDragTarget}
      draggable={draggable}
      onClick={() => onClick(view)}
      onContextMenu={(e) => onContextMenu(e, project)}
      onDragStart={onDragStart}
      onDragEnd={onDragEnd}
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
