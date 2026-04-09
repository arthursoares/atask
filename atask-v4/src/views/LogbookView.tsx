import { useLogbook, reopenTask } from "../store";
import CheckboxCircle from "../components/CheckboxCircle";
import TagPill from "../components/TagPill";
import DateGroupHeader from "../components/DateGroupHeader";
import EmptyState from "../components/EmptyState";
import type { Task } from "../types";

const LogbookIcon = (
  <svg viewBox="0 0 48 48" style={{ width: 48, height: 48 }}>
    <path d="M12 6h24l3 12-15 9-15-9z" fill="none" stroke="currentColor" strokeWidth="2" />
    <path
      d="M9 18v18c0 3 6 6 15 6s15-3 15-6V18"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
    />
  </svg>
);

const DAYS = ["Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"];
const MONTHS = ["Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"];

interface DateGroup {
  date: string;
  tasks: Task[];
}

function formatDate(dateStr: string): string {
  const d = new Date(dateStr + "T00:00:00");
  return `${DAYS[d.getDay()]}, ${MONTHS[d.getMonth()]} ${d.getDate()}`;
}

function groupByDate(tasks: Task[]): DateGroup[] {
  const groups: DateGroup[] = [];
  for (const task of tasks) {
    const date = task.completedAt ? task.completedAt.slice(0, 10) : "Unknown";
    const last = groups[groups.length - 1];
    if (last && last.date === date) {
      last.tasks.push(task);
    } else {
      groups.push({ date, tasks: [task] });
    }
  }
  return groups;
}

function relativeTime(isoStr: string): string {
  const now = Date.now();
  const then = new Date(isoStr).getTime();
  const diffMs = now - then;
  const diffMin = Math.floor(diffMs / 60000);
  const diffHr = Math.floor(diffMs / 3600000);
  const diffDay = Math.floor(diffMs / 86400000);

  if (diffMin < 1) return "Just now";
  if (diffMin < 60) return `${diffMin}m ago`;
  if (diffHr < 24) return `${diffHr}h ago`;
  if (diffDay === 1) return "Yesterday";

  const d = new Date(isoStr);
  return `${MONTHS[d.getMonth()]} ${d.getDate()}`;
}

function LogbookRow({ task }: { task: Task }) {
  const isCompleted = task.status === 1;
  const isCancelled = task.status === 2;

  return (
    <div className="task-item logbook-row">
      <CheckboxCircle
        checked={isCompleted}
        cancelled={isCancelled}
        onChange={() => reopenTask(task.id)}
      />
      <span className={`task-title ${isCompleted ? "completed" : ""}`}>{task.title}</span>
      {isCancelled && <TagPill label="Cancelled" variant="cancelled" />}
      <span className="task-meta">{task.completedAt ? relativeTime(task.completedAt) : ""}</span>
      <button className="reopen-btn" onClick={() => reopenTask(task.id)}>
        Reopen
      </button>
    </div>
  );
}

export default function LogbookView() {
  const tasks = useLogbook();
  const groups = groupByDate(tasks);

  return (
    <div>
      {groups.length === 0 ? (
        <EmptyState icon={LogbookIcon} text="No completed tasks" hint="Completed tasks across every view show up here." />
      ) : (
        groups.map((group) => (
          <div className="date-group" key={group.date}>
            <DateGroupHeader date={formatDate(group.date)} />
            {group.tasks.map((task) => (
              <LogbookRow key={task.id} task={task} />
            ))}
          </div>
        ))
      )}
    </div>
  );
}
