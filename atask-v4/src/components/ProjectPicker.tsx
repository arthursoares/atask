import { useEffect, useRef } from 'react';
import { useActiveProjects, updateTask } from '../store';

interface ProjectPickerProps {
  taskId: string;
  onClose: () => void;
}

const popoverStyle: React.CSSProperties = {
  position: 'absolute',
  top: '100%',
  left: 0,
  marginTop: 6,
  background: 'var(--canvas-elevated)',
  border: '1px solid var(--border-strong)',
  borderRadius: 'var(--radius-lg)',
  boxShadow: 'var(--shadow-popover)',
  minWidth: 200,
  padding: 0,
  zIndex: 50,
  overflow: 'hidden',
  userSelect: 'none',
};

const rowStyle: React.CSSProperties = {
  display: 'flex',
  alignItems: 'center',
  gap: 'var(--sp-3)',
  padding: '5px var(--sp-4)',
  fontSize: 'var(--text-base)',
  color: 'var(--ink-primary)',
  cursor: 'pointer',
};

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

  const handleSelect = (projectId: string | null) => {
    updateTask({ id: taskId, projectId });
    onClose();
  };

  const hoverOn = (e: React.MouseEvent<HTMLElement>) => {
    (e.currentTarget as HTMLElement).style.background = 'var(--sidebar-hover)';
  };
  const hoverOff = (e: React.MouseEvent<HTMLElement>) => {
    (e.currentTarget as HTMLElement).style.background = '';
  };

  return (
    <div style={popoverStyle} ref={popoverRef}>
      {/* Header */}
      <div
        style={{
          fontSize: 'var(--text-xs)',
          fontWeight: 700,
          color: 'var(--ink-tertiary)',
          padding: 'var(--sp-3) var(--sp-4) var(--sp-2)',
          textAlign: 'center',
        }}
      >
        Move to Project
      </div>
      <div style={{ height: 1, background: 'var(--separator)' }} />

      {/* Inbox option */}
      <div
        style={rowStyle}
        onClick={() => handleSelect(null)}
        onMouseEnter={hoverOn}
        onMouseLeave={hoverOff}
      >
        <span style={{ color: 'var(--ink-quaternary)' }}>Inbox (No Project)</span>
      </div>

      {/* Project list */}
      {projects.map((project) => (
        <div
          key={project.id}
          style={rowStyle}
          onClick={() => handleSelect(project.id)}
          onMouseEnter={hoverOn}
          onMouseLeave={hoverOff}
        >
          <span
            style={{
              width: 8,
              height: 8,
              borderRadius: '50%',
              background: project.color || 'var(--accent)',
              flexShrink: 0,
              display: 'inline-block',
            }}
          />
          <span>{project.title}</span>
        </div>
      ))}
    </div>
  );
}
