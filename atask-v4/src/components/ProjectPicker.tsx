import { useEffect, useRef } from 'react';
import { useActiveProjects, updateTask } from '../store';
import { PopoverPanel } from '../ui';

interface ProjectPickerProps {
  taskId: string;
  onClose: () => void;
}

export default function ProjectPicker({ taskId, onClose }: ProjectPickerProps) {
  const projects = useActiveProjects();

  const popoverRef = useRef<HTMLDivElement>(null);

  // Click-outside to close
  useEffect(() => {
    const handleMouseDown = (e: MouseEvent) => {
      if (popoverRef.current && !popoverRef.current.contains(e.target as Node)) {
        onClose();
      }
    };
    document.addEventListener('mousedown', handleMouseDown);
    return () => document.removeEventListener('mousedown', handleMouseDown);
  }, [onClose]);

  // Capture-phase Esc so the picker swallows the key before a containing
  // DetailPanel's Esc handler closes the whole panel.
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        e.stopPropagation();
        onClose();
      }
    };
    document.addEventListener('keydown', handleKeyDown, true);
    return () => document.removeEventListener('keydown', handleKeyDown, true);
  }, [onClose]);

  const handleSelect = (projectId: string | null) => {
    updateTask({ id: taskId, projectId });
    onClose();
  };

  return (
    <PopoverPanel title="Move to Project" popoverRef={popoverRef}>
      <button
        type="button"
        className="ui-picker-row"
        onClick={() => handleSelect(null)}
      >
        <span className="ui-picker-empty">Inbox (No Project)</span>
      </button>

      {projects.map((project) => (
        <button
          key={project.id}
          type="button"
          className="ui-picker-row"
          onClick={() => handleSelect(project.id)}
        >
          <span
            className="ui-picker-dot"
            style={{ background: project.color || 'var(--accent)' }}
          />
          <span className="ui-picker-label">{project.title}</span>
        </button>
      ))}
    </PopoverPanel>
  );
}
