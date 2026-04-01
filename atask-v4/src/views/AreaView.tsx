import { useStore } from '@nanostores/react';
import {
  $areas,
  $tasks,
  $selectedTaskId,
  $expandedTaskId,
  $selectedTaskIds,
  setActiveView,
  selectTask,
  openTaskEditor,
  closeTaskEditor,
  useActiveProjects,
} from '../store/index';
import TaskRow from '../components/TaskRow';
import TaskInlineEditor from '../components/TaskInlineEditor';
import EmptyState from '../components/EmptyState';
import ProgressBar from '../components/ProgressBar';

const AreaIcon = (
  <svg viewBox="0 0 48 48" style={{ width: 48, height: 48 }}>
    <rect x="6" y="6" width="36" height="36" rx="6" fill="none" stroke="currentColor" strokeWidth="2" />
    <line x1="6" y1="20" x2="42" y2="20" stroke="currentColor" strokeWidth="2" />
    <line x1="20" y1="20" x2="20" y2="42" stroke="currentColor" strokeWidth="2" />
  </svg>
);

interface AreaViewProps {
  areaId: string;
}

export default function AreaView({ areaId }: AreaViewProps) {
  const areas = useStore($areas);
  const tasks = useStore($tasks);
  const projects = useActiveProjects();
  const selectedTaskId = useStore($selectedTaskId);
  const expandedTaskId = useStore($expandedTaskId);
  const selectedTaskIds = useStore($selectedTaskIds);

  const area = areas.find((a) => a.id === areaId);
  const areaProjects = projects.filter((p) => p.areaId === areaId);
  const areaTasks = tasks.filter(
    (t) => t.areaId === areaId && t.status === 0 && !t.projectId,
  );

  // Stats
  const allAreaTasks = tasks.filter((t) => t.areaId === areaId || areaProjects.some((p) => p.id === t.projectId));
  const totalTasks = allAreaTasks.length;
  const completedTasks = allAreaTasks.filter((t) => t.status === 1).length;

  if (!area) {
    return <EmptyState icon={AreaIcon} text="Area not found" />;
  }

  return (
    <div>
      {/* Area overview stats */}
      <div style={{ marginBottom: "var(--sp-4)" }}>
        <ProgressBar completed={completedTasks} total={totalTasks} />
        <div style={{
          display: "flex",
          gap: "var(--sp-5)",
          marginTop: "var(--sp-3)",
          fontSize: "var(--text-sm)",
          color: "var(--ink-tertiary)",
        }}>
          <span>{areaProjects.length} project{areaProjects.length !== 1 ? "s" : ""}</span>
          <span>{areaTasks.length} direct task{areaTasks.length !== 1 ? "s" : ""}</span>
        </div>
      </div>

      {/* Projects in this area */}
      {areaProjects.length > 0 && (
        <div style={{ marginBottom: "var(--sp-4)" }}>
          <div style={{
            fontSize: "var(--text-xs)",
            fontWeight: 700,
            color: "var(--ink-tertiary)",
            textTransform: "uppercase",
            letterSpacing: "0.5px",
            marginBottom: "var(--sp-2)",
          }}>
            Projects
          </div>
          {areaProjects.map((project) => {
            const projectTasks = tasks.filter((t) => t.projectId === project.id && t.status === 0);
            return (
              <div
                key={project.id}
                className="sidebar-item"
                style={{ padding: "var(--sp-2) var(--sp-3)", cursor: "pointer" }}
                onClick={() => setActiveView(`project-${project.id}`)}
              >
                <span
                  style={{
                    width: 8,
                    height: 8,
                    borderRadius: "50%",
                    background: project.color || "var(--accent)",
                    flexShrink: 0,
                  }}
                />
                <span style={{ flex: 1 }}>{project.title}</span>
                <span style={{ fontSize: "var(--text-xs)", color: "var(--ink-tertiary)" }}>
                  {projectTasks.length}
                </span>
              </div>
            );
          })}
        </div>
      )}

      {/* Direct tasks (tasks assigned to area but not in a project) */}
      {areaTasks.length > 0 && (
        <div>
          <div style={{
            fontSize: "var(--text-xs)",
            fontWeight: 700,
            color: "var(--ink-tertiary)",
            textTransform: "uppercase",
            letterSpacing: "0.5px",
            marginBottom: "var(--sp-2)",
          }}>
            Tasks
          </div>
          {areaTasks.map((task) =>
            expandedTaskId === task.id ? (
              <TaskInlineEditor
                key={task.id}
                task={task}
                onClose={closeTaskEditor}
              />
            ) : (
              <TaskRow
                key={task.id}
                task={task}
                isSelected={selectedTaskId === task.id}
                isMultiSelected={selectedTaskIds.has(task.id)}
                onClick={() => selectTask(task.id)}
                onDoubleClick={() => openTaskEditor(task.id)}
              />
            ),
          )}
        </div>
      )}

      {areaProjects.length === 0 && areaTasks.length === 0 && (
        <EmptyState icon={AreaIcon} text="No projects or tasks in this area" />
      )}
    </div>
  );
}
