import type { Project, Task } from '../../types';

interface AreaProjectListProps {
  projects: Project[];
  tasks: Task[];
  onOpenProject: (projectId: string) => void;
}

export default function AreaProjectList({
  projects,
  tasks,
  onOpenProject,
}: AreaProjectListProps) {
  if (projects.length === 0) return null;

  return (
    <div className="area-section">
      <div className="area-section-label">Projects</div>
      {projects.map((project) => {
        const projectTaskCount = tasks.filter(
          (task) => task.projectId === project.id && task.status === 0,
        ).length;

        return (
          <div
            key={project.id}
            className="sidebar-item area-project-row"
            onClick={() => onOpenProject(project.id)}
          >
            <span
              className="area-project-dot"
              style={{ background: project.color || 'var(--accent)' }}
            />
            <span className="area-project-title">{project.title}</span>
            <span className="area-project-count">{projectTaskCount}</span>
          </div>
        );
      })}
    </div>
  );
}
