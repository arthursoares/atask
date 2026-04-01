import SectionHeader from '../../components/SectionHeader';
import ContextMenu, { type MenuItem } from '../../components/ContextMenu';
import { Field } from '../../ui';
import ProjectTaskList from './ProjectTaskList';
import type { ReorderMove, Section, Task } from '../../types';

interface ProjectSectionBlockProps {
  section: Section;
  tasks: Task[];
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
}

export default function ProjectSectionBlock({
  section,
  tasks,
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
}: ProjectSectionBlockProps) {
  return (
    <div>
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
        <div onContextMenu={(event) => onContextMenu(event, section.id)}>
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
