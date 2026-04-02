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
export { default as DropSlot } from "./DropSlot";

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
  hideProjectPill,
}: {
  task: Task;
  project: Project | null;
  taskTags: Tag[];
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
