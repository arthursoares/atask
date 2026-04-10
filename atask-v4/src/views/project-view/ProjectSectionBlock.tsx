import { useState } from 'react';
import SectionHeader from '../../components/SectionHeader';
import ContextMenu, { type MenuItem } from '../../components/ContextMenu';
import { Field } from '../../ui';
import ProjectTaskList from './ProjectTaskList';
import type { ReorderMove, Section, Task } from '../../types';
import { useStore } from '@nanostores/react';
import { $taskPointerDrag } from '../../store/ui';

interface ProjectSectionBlockProps {
  section: Section;
  tasks: Task[];
  projectId: string;
  expandedTaskId: string | null;
  selectedTaskId: string | null;
  selectedTaskIds: Set<string>;
  isRenaming: boolean;
  renamingValue: string;
  renameInputRef: React.RefObject<HTMLInputElement | null>;
  menuPosition: { x: number; y: number } | null;
  onContextMenu: (event: React.MouseEvent, sectionId: string) => void;
  onRenameChange: (value: string) => void;
  onRenameCommit: (sectionId: string) => void;
  onRenameCancel: () => void;
  onToggleCollapsed: (sectionId: string) => void;
  onSelectTask: (id: string) => void;
  onExpandTask: (id: string) => void;
  onCloseExpandedTask: () => void;
  onCreateTask: (title: string, sectionId: string) => void;
  onCloseMenu: () => void;
  buildMenuItems: (sectionId: string) => MenuItem[];
  onReorderTasks: (moves: ReorderMove[]) => Promise<void>;
  onTaskDrop?: (taskId: string, sectionId: string) => void;
}

export default function ProjectSectionBlock({
  section,
  tasks,
  projectId,
  expandedTaskId,
  selectedTaskId,
  selectedTaskIds,
  isRenaming,
  renamingValue,
  renameInputRef,
  menuPosition,
  onContextMenu,
  onRenameChange,
  onRenameCommit,
  onRenameCancel,
  onToggleCollapsed,
  onSelectTask,
  onExpandTask,
  onCloseExpandedTask,
  onCreateTask,
  onCloseMenu,
  buildMenuItems,
  onReorderTasks,
  onTaskDrop,
}: ProjectSectionBlockProps) {
  const [isDropTarget, setIsDropTarget] = useState(false);
  const taskDrag = useStore($taskPointerDrag);

  const handlePointerEnter = () => {
    if (taskDrag.activeTaskId && onTaskDrop) {
      setIsDropTarget(true);
    }
  };

  const handlePointerLeave = () => {
    setIsDropTarget(false);
  };

  const handlePointerUp = () => {
    if (taskDrag.activeTaskId && onTaskDrop) {
      onTaskDrop(taskDrag.activeTaskId, section.id);
    }
    setIsDropTarget(false);
  };

  return (
    <div
      // Wrap the whole section block (header + task list) as the section
      // drop target so dropping a task anywhere inside this block — header
      // OR tasks area — resolves to this section.id via the hook's
      // closest-ancestor walk. Previously only the header wrapper had
      // these attributes, so releasing over the task list of another
      // section fell through to a null target and the cross-list drop
      // never fired.
      data-sidebar-item-kind="section"
      data-sidebar-item-id={section.id}
    >
      {isRenaming ? (
        <div className="project-section-rename">
          <Field
            ref={renameInputRef}
            value={renamingValue}
            className="project-section-rename-input"
            onChange={(event) => onRenameChange(event.target.value)}
            onKeyDown={(event) => {
              if (event.key === 'Enter') {
                event.preventDefault();
                onRenameCommit(section.id);
              } else if (event.key === 'Escape') {
                onRenameCancel();
              }
            }}
            onBlur={() => onRenameCommit(section.id)}
          />
        </div>
      ) : (
        <div
          onContextMenu={(event) => onContextMenu(event, section.id)}
          onPointerEnter={handlePointerEnter}
          onPointerLeave={handlePointerLeave}
          onPointerUp={handlePointerUp}
          data-sidebar-item-kind="section"
          data-sidebar-item-id={section.id}
          style={isDropTarget ? { background: 'var(--accent-subtle)', borderRadius: 'var(--radius-md)' } : undefined}
        >
          <SectionHeader
            title={section.title}
            count={tasks.length}
            collapsible
            collapsed={section.collapsed}
            onToggle={() => onToggleCollapsed(section.id)}
          />
        </div>
      )}

      {!section.collapsed && (
        <ProjectTaskList
          tasks={tasks}
          projectId={projectId}
          listId={`task-section:${section.id}`}
          expandedTaskId={expandedTaskId}
          selectedTaskId={selectedTaskId}
          selectedTaskIds={selectedTaskIds}
          onSelectTask={onSelectTask}
          onExpandTask={onExpandTask}
          onCloseExpandedTask={onCloseExpandedTask}
          onCreateTask={(title) => onCreateTask(title, section.id)}
          onReorderTasks={onReorderTasks}
        />
      )}

      {menuPosition && (
        <ContextMenu
          items={buildMenuItems(section.id)}
          position={menuPosition}
          onClose={onCloseMenu}
        />
      )}
    </div>
  );
}
