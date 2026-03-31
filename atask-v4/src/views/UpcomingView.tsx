import { useStore } from "@nanostores/react";
import { useUpcoming, $selectedTaskId, $expandedTaskId } from "../store/index";
import TaskRow from "../components/TaskRow";
import TaskInlineEditor from "../components/TaskInlineEditor";
import DateGroupHeader from "../components/DateGroupHeader";
import EmptyState from "../components/EmptyState";

const CalendarIcon = (
  <svg viewBox="0 0 48 48" style={{ width: 48, height: 48 }}>
    <rect
      x="6"
      y="9"
      width="36"
      height="33"
      rx="6"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
    />
    <line x1="6" y1="21" x2="42" y2="21" stroke="currentColor" strokeWidth="2" />
    <line x1="15" y1="3" x2="15" y2="12" stroke="currentColor" strokeWidth="2" />
    <line x1="33" y1="3" x2="33" y2="12" stroke="currentColor" strokeWidth="2" />
  </svg>
);

const DAYS = ["Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"];
const MONTHS = ["Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"];

function formatDate(dateStr: string): string {
  const d = new Date(dateStr + "T00:00:00");
  return `${DAYS[d.getDay()]}, ${MONTHS[d.getMonth()]} ${d.getDate()}`;
}

function getRelativeDate(dateStr: string): string | undefined {
  const today = new Date();
  today.setHours(0, 0, 0, 0);
  const target = new Date(dateStr + "T00:00:00");
  target.setHours(0, 0, 0, 0);
  const diffDays = Math.round((target.getTime() - today.getTime()) / (1000 * 60 * 60 * 24));

  if (diffDays === 1) return "Tomorrow";
  if (diffDays > 1 && diffDays <= 7) return `In ${diffDays} days`;
  return undefined;
}

export default function UpcomingView() {
  const groups = useUpcoming();
  const selectedTaskId = useStore($selectedTaskId);
  const expandedTaskId = useStore($expandedTaskId);

  return (
    <div>
      {groups.length === 0 ? (
        <EmptyState icon={CalendarIcon} text="Nothing scheduled" />
      ) : (
        groups.map((group) => (
          <div className="date-group" key={group.date}>
            <DateGroupHeader date={formatDate(group.date)} relative={getRelativeDate(group.date)} />
            {group.tasks.map((task) =>
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
                  onClick={() => $selectedTaskId.set(task.id)}
                  onDoubleClick={() => $expandedTaskId.set(task.id)}
                />
              ),
            )}
          </div>
        ))
      )}
    </div>
  );
}
