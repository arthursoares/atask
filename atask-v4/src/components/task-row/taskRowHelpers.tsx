import TagPill from "../TagPill";
import type { MenuItem } from "../ContextMenu";
import type { Project, Tag, Task } from "../../types";
import { todayLocal } from "../../lib/dates";
import {
  cancelTask,
  completeTask,
  deleteTask,
  duplicateTask,
  reopenTask,
  updateTask,
} from "../../store";

export function formatDeadline(deadline: string): string {
  const today = new Date();
  today.setHours(0, 0, 0, 0);

  const d = new Date(deadline);
  d.setHours(0, 0, 0, 0);

  const diffDays = Math.round((d.getTime() - today.getTime()) / (1000 * 60 * 60 * 24));

  if (diffDays < 0) return "Overdue";
  if (diffDays === 0) return "Today";
  if (diffDays === 1) return "Tomorrow";

  return d.toLocaleDateString("en-US", { month: "short", day: "numeric" });
}

export function buildTaskContextMenuItems(task: Task, isCompleted: boolean, isCancelled: boolean): MenuItem[] {
  return [
    {
      label: isCompleted || isCancelled ? "Reopen" : "Complete",
      onClick: () => {
        if (isCompleted || isCancelled) {
          reopenTask(task.id);
        } else {
          completeTask(task.id);
        }
      },
    },
    {
      label: "Cancel",
      onClick: () => cancelTask(task.id),
    },
    { separator: true },
    {
      label: "Today",
      shortcut: "⌘T",
      onClick: () => {
        const today = todayLocal();
        updateTask({ id: task.id, schedule: 1, startDate: today });
      },
    },
    {
      label: "Evening",
      shortcut: "⌘E",
      onClick: () => {
        const today = todayLocal();
        updateTask({ id: task.id, schedule: 1, timeSlot: "evening", startDate: today });
      },
    },
    {
      label: "Someday",
      shortcut: "⌘O",
      onClick: () => updateTask({ id: task.id, schedule: 2 }),
    },
    {
      label: "Inbox",
      onClick: () => updateTask({ id: task.id, schedule: 0 }),
    },
    { separator: true },
    {
      label: "Duplicate",
      shortcut: "⌘D",
      onClick: () => duplicateTask(task.id),
    },
    {
      label: "Delete",
      shortcut: "⌫",
      danger: true,
      onClick: () => deleteTask(task.id),
    },
  ];
}

export function TaskMeta({
  task,
  project,
  taskTags,
  checklistCount,
  hideProjectPill,
}: {
  task: Task;
  project: Project | null;
  taskTags: Tag[];
  checklistCount?: { done: number; total: number };
  hideProjectPill?: boolean;
}) {
  const metaItems: React.ReactNode[] = [];

  if (project && !hideProjectPill) {
    metaItems.push(
      <span key="project" className="task-project-pill">
        <span
          className="task-project-dot"
          style={{ background: project.color || "var(--accent)" }}
        />
        {project.title}
      </span>,
    );
  }

  if (task.deadline) {
    if (metaItems.length > 0) {
      metaItems.push(<span key="sep-deadline" className="task-meta-sep">·</span>);
    }
    metaItems.push(
      <span key="deadline" className="task-deadline">
        {formatDeadline(task.deadline)}
      </span>,
    );
  }

  // Checklist count badge — per design spec (2026-03-29-v4-tauri-react-
  // design.md line 310). Rendered as a small count "2/5" with a check
  // glyph so users can see progress inline without opening the editor.
  // Tasks without checklist items get nothing (no count badge = no noise).
  if (checklistCount && checklistCount.total > 0) {
    if (metaItems.length > 0) {
      metaItems.push(<span key="sep-checklist" className="task-meta-sep">·</span>);
    }
    const allDone = checklistCount.done === checklistCount.total;
    metaItems.push(
      <span
        key="checklist"
        className={`task-checklist-count${allDone ? " done" : ""}`}
        aria-label={`Checklist: ${checklistCount.done} of ${checklistCount.total} complete`}
      >
        <svg viewBox="0 0 12 12" width="10" height="10" aria-hidden="true">
          <rect x="1" y="1" width="10" height="10" rx="1.5" fill="none" stroke="currentColor" strokeWidth="1.2" />
          {allDone && (
            <polyline
              points="3 6.2 5 8.2 9 4.2"
              fill="none"
              stroke="currentColor"
              strokeWidth="1.4"
              strokeLinecap="round"
              strokeLinejoin="round"
            />
          )}
        </svg>
        <span>{checklistCount.done}/{checklistCount.total}</span>
      </span>,
    );
  }

  if (taskTags.length > 0) {
    for (const tag of taskTags) {
      if (metaItems.length > 0) {
        metaItems.push(<span key={`sep-${tag.id}`} className="task-meta-sep">·</span>);
      }
      metaItems.push(<TagPill key={tag.id} label={tag.title} variant="default" />);
    }
  }

  if (metaItems.length === 0) return null;

  return <div className="task-meta">{metaItems}</div>;
}
