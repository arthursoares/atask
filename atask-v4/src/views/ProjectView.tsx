import { useRef, useState } from 'react';
import { useStore } from '@nanostores/react';
import {
  useTasksForProject,
  useSectionsForProject,
  $projects,
  $selectedTaskId,
  $expandedTaskId,
  $selectedTaskIds,
  $tasks,
  selectTask,
  openTaskEditor,
  closeTaskEditor,
  createTask,
  createSection,
  updateSection,
  deleteSection,
  toggleSectionCollapsed,
  toggleSectionArchived,
  moveTaskToSection,
  reorderTasks,
} from '../store/index';
import ProgressBar from '../components/ProgressBar';
import EmptyState from '../components/EmptyState';
import type { MenuItem } from '../components/ContextMenu';
import { Button } from '../ui';
import ProjectSectionBlock from './project-view/ProjectSectionBlock';
import ProjectTaskList from './project-view/ProjectTaskList';

const ProjectIcon = (
  <svg viewBox="0 0 48 48" style={{ width: 48, height: 48 }}>
    <rect
      x="6"
      y="12"
      width="36"
      height="27"
      rx="4"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
    />
    <path
      d="M6 12V9a3 3 0 013-3h10l3 6H42"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
    />
  </svg>
);

interface ProjectViewProps {
  projectId: string;
}

export default function ProjectView({ projectId }: ProjectViewProps) {
  const projects = useStore($projects);
  const project = projects.find((p) => p.id === projectId);
  const allProjectTasks = useTasksForProject(projectId);
  const sections = useSectionsForProject(projectId);

  const selectedTaskId = useStore($selectedTaskId);
  const expandedTaskId = useStore($expandedTaskId);
  const selectedTaskIds = useStore($selectedTaskIds);

  const allTasks = useStore($tasks);
  const projectTasks = allTasks.filter((t) => t.projectId === projectId);
  const completedCount = projectTasks.filter((t) => t.status === 1).length;
  const totalCount = projectTasks.length;

  const sectionlessTasks = allProjectTasks
    .filter((t) => t.sectionId === null)
    .sort((a, b) => a.index - b.index);

  // Section context menu state
  const [sectionMenu, setSectionMenu] = useState<{
    sectionId: string;
    x: number;
    y: number;
  } | null>(null);
  const [renamingSection, setRenamingSection] = useState<string | null>(null);
  const [renamingValue, setRenamingValue] = useState('');
  const renameInputRef = useRef<HTMLInputElement>(null);

  const handleSectionContextMenu = (
    e: React.MouseEvent,
    sectionId: string,
  ) => {
    e.preventDefault();
    e.stopPropagation();
    setSectionMenu({ sectionId, x: e.clientX, y: e.clientY });
  };

  const handleRenameCommit = async (sectionId: string) => {
    const title = renamingValue.trim();
    if (title) {
      await updateSection({ id: sectionId, title });
    }
    closeSectionRename();
  };

  const handleSectionCreate = async (title: string, sectionId: string) => {
    const task = await createTask(title);
    await moveTaskToSection(task.id, sectionId);
  };

  const closeSectionRename = () => {
    setRenamingSection(null);
    setRenamingValue('');
  };

  const buildSectionMenuItems = (sectionId: string): MenuItem[] => {
    const section = sections.find((s) => s.id === sectionId);
    if (!section) return [];
    return [
      {
        label: 'Rename',
        onClick: () => {
          setRenamingSection(sectionId);
          setRenamingValue(section.title);
          setTimeout(() => renameInputRef.current?.focus(), 0);
        },
      },
      {
        label: section.archived ? 'Unarchive' : 'Archive',
        onClick: () => toggleSectionArchived(sectionId),
      },
      { separator: true },
      {
        label: 'Delete',
        danger: true,
        onClick: () => deleteSection(sectionId),
      },
    ];
  };

  if (!project) {
    return <EmptyState icon={ProjectIcon} text="Project not found" />;
  }

  return (
    <div>
      {/* Toolbar extras */}
      <div className="project-toolbar-row">
        <ProgressBar completed={completedCount} total={totalCount} />
        <Button
          variant="ghost"
          size="sm"
          onClick={() => createSection({ title: 'New Section', projectId })}
        >
          + Add Section
        </Button>
      </div>

      {/* Sectionless tasks */}
      <ProjectTaskList
        tasks={sectionlessTasks}
        projectId={project.id}
        expandedTaskId={expandedTaskId}
        selectedTaskId={selectedTaskId}
        selectedTaskIds={selectedTaskIds}
        onSelectTask={selectTask}
        onExpandTask={openTaskEditor}
        onCloseExpandedTask={closeTaskEditor}
        onCreateTask={(title) => createTask(title)}
        onReorderTasks={reorderTasks}
        onTaskDrop={async (taskId) => {
          await moveTaskToSection(taskId, null);
        }}
      />

      {/* Sections */}
      {sections.map((section) => {
        const sectionTasks = allProjectTasks
          .filter((t) => t.sectionId === section.id)
          .sort((a, b) => a.index - b.index);

        return (
          <ProjectSectionBlock
            key={section.id}
            section={section}
            tasks={sectionTasks}
            projectId={project.id}
            expandedTaskId={expandedTaskId}
            selectedTaskId={selectedTaskId}
            selectedTaskIds={selectedTaskIds}
            isRenaming={renamingSection === section.id}
            renamingValue={renamingValue}
            renameInputRef={renameInputRef}
            menuPosition={
              sectionMenu?.sectionId === section.id
                ? { x: sectionMenu.x, y: sectionMenu.y }
                : null
            }
            onContextMenu={handleSectionContextMenu}
            onRenameChange={setRenamingValue}
            onRenameCommit={handleRenameCommit}
            onRenameCancel={closeSectionRename}
            onToggleCollapsed={toggleSectionCollapsed}
            onSelectTask={selectTask}
            onExpandTask={openTaskEditor}
            onCloseExpandedTask={closeTaskEditor}
            onCreateTask={handleSectionCreate}
            onCloseMenu={() => setSectionMenu(null)}
            buildMenuItems={buildSectionMenuItems}
            onReorderTasks={reorderTasks}
            onTaskDrop={async (taskId, sectionId) => {
              await moveTaskToSection(taskId, sectionId);
            }}
          />
        );
      })}

      {/* Empty state: only show if truly empty */}
      {sectionlessTasks.length === 0 && sections.length === 0 && (
        <EmptyState icon={ProjectIcon} text="No tasks in this project" hint={<>Press <kbd>⌘N</kbd> to add one.</>} />
      )}
    </div>
  );
}
