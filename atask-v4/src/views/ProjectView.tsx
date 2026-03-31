import { useState, useRef } from "react";
import { useStore } from "@nanostores/react";
import {
  useTasksForProject,
  useSectionsForProject,
  $projects,
  $selectedTaskId,
  $expandedTaskId,
  $selectedTaskIds,
  $tasks,
  createTask,
  createSection,
  updateSection,
  deleteSection,
  toggleSectionCollapsed,
  toggleSectionArchived,
  moveTaskToSection,
  reorderTasks,
} from "../store/index";
import TaskRow from "../components/TaskRow";
import TaskInlineEditor from "../components/TaskInlineEditor";
import NewTaskRow from "../components/NewTaskRow";
import SectionHeader from "../components/SectionHeader";
import ProgressBar from "../components/ProgressBar";
import Button from "../components/Button";
import EmptyState from "../components/EmptyState";
import ContextMenu, { type MenuItem } from "../components/ContextMenu";
import useDragReorder from "../hooks/useDragReorder";
import type { Task } from "../types";

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

interface SectionTaskListProps {
  tasks: Task[];
  expandedTaskId: string | null;
  selectedTaskId: string | null;
  selectedTaskIds: Set<string>;
  setSelectedTaskId: (id: string) => void;
  setExpandedTaskId: (id: string | null) => void;
  onCreateTask: (title: string) => void;
}

function SectionTaskList({
  tasks,
  expandedTaskId,
  selectedTaskId,
  selectedTaskIds,
  setSelectedTaskId,
  setExpandedTaskId,
  onCreateTask,
}: SectionTaskListProps) {
  const { dragState, getDragHandlers, getDropHandlers } = useDragReorder(tasks, reorderTasks);

  return (
    <>
      {tasks.map((task, index) =>
        expandedTaskId === task.id ? (
          <TaskInlineEditor
            key={task.id}
            task={task}
            onClose={() => setExpandedTaskId(null)}
          />
        ) : (
          <TaskRow
            key={task.id}
            task={task}
            isSelected={selectedTaskId === task.id}
            isMultiSelected={selectedTaskIds.has(task.id)}
            taskList={tasks}
            hideProjectPill
            onClick={() => setSelectedTaskId(task.id)}
            onDoubleClick={() => setExpandedTaskId(task.id)}
            dragHandlers={getDragHandlers(task.id)}
            dropHandlers={getDropHandlers(index)}
            isDragOver={dragState.dropIndex === index && dragState.dragId !== task.id}
          />
        ),
      )}
      <NewTaskRow onCreate={onCreateTask} />
    </>
  );
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

  const {
    dragState: sectionlessDragState,
    getDragHandlers: getSectionlessDragHandlers,
    getDropHandlers: getSectionlessDropHandlers,
  } = useDragReorder(sectionlessTasks, reorderTasks);

  // Section context menu state
  const [sectionMenu, setSectionMenu] = useState<{
    sectionId: string;
    x: number;
    y: number;
  } | null>(null);
  const [renamingSection, setRenamingSection] = useState<string | null>(null);
  const [renamingValue, setRenamingValue] = useState("");
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
    setRenamingSection(null);
    setRenamingValue("");
  };

  const handleSectionCreate = async (title: string, sectionId: string) => {
    const task = await createTask(title);
    await moveTaskToSection(task.id, sectionId);
  };

  const buildSectionMenuItems = (sectionId: string): MenuItem[] => {
    const section = sections.find((s) => s.id === sectionId);
    if (!section) return [];
    return [
      {
        label: "Rename",
        onClick: () => {
          setRenamingSection(sectionId);
          setRenamingValue(section.title);
          // Focus input on next tick
          setTimeout(() => renameInputRef.current?.focus(), 0);
        },
      },
      {
        label: section.archived ? "Unarchive" : "Archive",
        onClick: () => toggleSectionArchived(sectionId),
      },
      { separator: true },
      {
        label: "Delete",
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
      <div
        style={{
          display: "flex",
          alignItems: "center",
          justifyContent: "space-between",
          marginBottom: "var(--sp-4)",
        }}
      >
        <ProgressBar completed={completedCount} total={totalCount} />
        <Button
          variant="ghost"
          size="sm"
          onClick={() => createSection({ title: "New Section", projectId })}
        >
          + Add Section
        </Button>
      </div>

      {/* Sectionless tasks */}
      {sectionlessTasks.map((task, index) =>
        expandedTaskId === task.id ? (
          <TaskInlineEditor
            key={task.id}
            task={task}
            onClose={() => $expandedTaskId.set(null)}
          />
        ) : (
          <TaskRow
            key={task.id}
            task={task}
            isSelected={selectedTaskId === task.id}
            isMultiSelected={selectedTaskIds.has(task.id)}
            taskList={sectionlessTasks}
            hideProjectPill
            onClick={() => $selectedTaskId.set(task.id)}
            onDoubleClick={() => $expandedTaskId.set(task.id)}
            dragHandlers={getSectionlessDragHandlers(task.id)}
            dropHandlers={getSectionlessDropHandlers(index)}
            isDragOver={sectionlessDragState.dropIndex === index && sectionlessDragState.dragId !== task.id}
          />
        ),
      )}
      <NewTaskRow onCreate={(title) => createTask(title)} />

      {/* Sections */}
      {sections.map((section) => {
        const sectionTasks = allProjectTasks
          .filter((t) => t.sectionId === section.id)
          .sort((a, b) => a.index - b.index);

        return (
          <div key={section.id}>
            {renamingSection === section.id ? (
              <div style={{ padding: "var(--sp-1) 0" }}>
                <input
                  ref={renameInputRef}
                  value={renamingValue}
                  onChange={(e) => setRenamingValue(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === "Enter") {
                      e.preventDefault();
                      handleRenameCommit(section.id);
                    } else if (e.key === "Escape") {
                      setRenamingSection(null);
                      setRenamingValue("");
                    }
                  }}
                  onBlur={() => handleRenameCommit(section.id)}
                  style={{
                    width: "100%",
                    background: "transparent",
                    border: "none",
                    borderBottom: "1px solid var(--accent)",
                    outline: "none",
                    fontSize: "var(--text-sm)",
                    fontWeight: 600,
                    color: "var(--ink-primary)",
                    padding: "var(--sp-1) 0",
                  }}
                />
              </div>
            ) : (
              <div onContextMenu={(e) => handleSectionContextMenu(e, section.id)}>
                <SectionHeader
                  title={section.title}
                  count={sectionTasks.length}
                  collapsible
                  collapsed={section.collapsed}
                  onToggle={() => toggleSectionCollapsed(section.id)}
                />
              </div>
            )}

            {!section.collapsed && (
              <SectionTaskList
                tasks={sectionTasks}
                expandedTaskId={expandedTaskId}
                selectedTaskId={selectedTaskId}
                selectedTaskIds={selectedTaskIds}
                setSelectedTaskId={(id) => $selectedTaskId.set(id)}
                setExpandedTaskId={(id) => $expandedTaskId.set(id)}
                onCreateTask={(title) => handleSectionCreate(title, section.id)}
              />
            )}
          </div>
        );
      })}

      {/* Empty state: only show if truly empty */}
      {sectionlessTasks.length === 0 && sections.length === 0 && (
        <EmptyState icon={ProjectIcon} text="No tasks in this project" />
      )}

      {/* Section context menu */}
      {sectionMenu && (
        <ContextMenu
          items={buildSectionMenuItems(sectionMenu.sectionId)}
          position={{ x: sectionMenu.x, y: sectionMenu.y }}
          onClose={() => setSectionMenu(null)}
        />
      )}
    </div>
  );
}
