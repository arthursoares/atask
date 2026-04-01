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
import EmptyState from '../components/EmptyState';
import ProgressBar from '../components/ProgressBar';
import AreaProjectList from './area-view/AreaProjectList';
import AreaTaskList from './area-view/AreaTaskList';

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
      <div className="area-summary">
        <ProgressBar completed={completedTasks} total={totalTasks} />
        <div className="area-summary-stats">
          <span>{areaProjects.length} project{areaProjects.length !== 1 ? 's' : ''}</span>
          <span>{areaTasks.length} direct task{areaTasks.length !== 1 ? 's' : ''}</span>
        </div>
      </div>

      <AreaProjectList
        projects={areaProjects}
        tasks={tasks}
        onOpenProject={(projectId) => setActiveView(`project-${projectId}`)}
      />

      <AreaTaskList
        tasks={areaTasks}
        selectedTaskId={selectedTaskId}
        selectedTaskIds={selectedTaskIds}
        expandedTaskId={expandedTaskId}
        onSelectTask={selectTask}
        onExpandTask={openTaskEditor}
        onCloseExpandedTask={closeTaskEditor}
      />

      {areaProjects.length === 0 && areaTasks.length === 0 && (
        <EmptyState icon={AreaIcon} text="No projects or tasks in this area" />
      )}
    </div>
  );
}
