import { useState, useEffect, useRef, useCallback } from "react";
import type { ReactNode } from "react";
import { todayLocal } from '../lib/dates';
import { useStore } from "@nanostores/react";
import {
  $showPalette,
  $selectedTaskId,
  $activeView,
  updateTask,
  completeTask,
  deleteTask,
  createTask,
  createProject,
} from "../store/index";
import {
  InboxIcon,
  TodayIcon,
  UpcomingIcon,
  SomedayIcon,
  LogbookIcon,
} from "./sidebar/SidebarIcons";

// Monochrome line icons matching the sidebar set — the palette previously
// mixed colored emoji into an otherwise line-icon design language.
function MoonIcon() {
  return (
    <svg viewBox="0 0 16 16">
      <path d="M13 9.5A5.5 5.5 0 0 1 6.5 3 5.5 5.5 0 1 0 13 9.5z" />
    </svg>
  );
}

function CheckIcon() {
  return (
    <svg viewBox="0 0 16 16">
      <polyline points="3 8.5 6.5 12 13 4.5" />
    </svg>
  );
}

function TrashIcon() {
  return (
    <svg viewBox="0 0 16 16">
      <path d="M3 4.5h10M6.5 4.5V3h3v1.5M4.5 4.5l.7 8.5h5.6l.7-8.5" />
    </svg>
  );
}

function PlusIcon() {
  return (
    <svg viewBox="0 0 16 16">
      <line x1="8" y1="3" x2="8" y2="13" />
      <line x1="3" y1="8" x2="13" y2="8" />
    </svg>
  );
}

interface Command {
  group: string;
  icon: ReactNode;
  label: string;
  shortcut?: string;
  action: () => void;
  keywords?: string[];
}

function matchesQuery(cmd: Command, query: string): boolean {
  if (!query) return true;
  const haystack = [cmd.label, ...(cmd.keywords ?? [])].join(" ").toLowerCase();
  const words = query.toLowerCase().split(/\s+/).filter(Boolean);
  return words.every((w) => haystack.includes(w));
}

export default function CommandPalette() {
  const showPalette = useStore($showPalette);
  const selectedTaskId = useStore($selectedTaskId);
  const setShowPalette = (v: boolean) => $showPalette.set(v);

  const [query, setQuery] = useState("");
  const [activeIndex, setActiveIndex] = useState(0);
  const inputRef = useRef<HTMLInputElement>(null);

  // Reset query and focus input when palette opens
  useEffect(() => {
    if (showPalette) {
      setQuery("");
      setActiveIndex(0);
      // Use a small timeout to ensure the element is rendered before focusing
      setTimeout(() => {
        inputRef.current?.focus();
      }, 0);
    }
  }, [showPalette]);

  // Reset active index when query changes
  useEffect(() => {
    setActiveIndex(0);
  }, [query]);

  const handleClose = useCallback(() => {
    setShowPalette(false);
    setQuery("");
  }, [setShowPalette]);

  // Build commands list
  const buildCommands = useCallback((): Command[] => {
    const commands: Command[] = [
      // Navigation group
      {
        group: "Navigation",
        icon: <InboxIcon />,
        label: "Go to Inbox",
        shortcut: "⌘1",
        keywords: ["inbox", "navigate"],
        action: () => {
          $activeView.set("inbox");
          handleClose();
        },
      },
      {
        group: "Navigation",
        icon: <TodayIcon />,
        label: "Go to Today",
        shortcut: "⌘2",
        keywords: ["today", "navigate"],
        action: () => {
          $activeView.set("today");
          handleClose();
        },
      },
      {
        group: "Navigation",
        icon: <UpcomingIcon />,
        label: "Go to Upcoming",
        shortcut: "⌘3",
        keywords: ["upcoming", "navigate", "scheduled"],
        action: () => {
          $activeView.set("upcoming");
          handleClose();
        },
      },
      {
        group: "Navigation",
        icon: <SomedayIcon />,
        label: "Go to Someday",
        shortcut: "⌘4",
        keywords: ["someday", "navigate", "later"],
        action: () => {
          $activeView.set("someday");
          handleClose();
        },
      },
      {
        group: "Navigation",
        icon: <LogbookIcon />,
        label: "Go to Logbook",
        shortcut: "⌘5",
        keywords: ["logbook", "navigate", "completed", "done"],
        action: () => {
          $activeView.set("logbook");
          handleClose();
        },
      },
    ];

    // Task Actions — only if a task is selected
    if (selectedTaskId) {
      const taskId = selectedTaskId;
      commands.push(
        {
          group: "Task Actions",
          icon: <TodayIcon />,
          label: "Schedule for Today",
          shortcut: "⌘T",
          keywords: ["schedule", "today", "task"],
          action: () => {
            const today = todayLocal();
            updateTask({ id: taskId, schedule: 1, startDate: today });
            handleClose();
          },
        },
        {
          group: "Task Actions",
          icon: <MoonIcon />,
          label: "Schedule for Evening",
          shortcut: "⌘E",
          keywords: ["schedule", "evening", "task", "tonight"],
          action: () => {
            const today = todayLocal();
            updateTask({ id: taskId, schedule: 1, timeSlot: "evening", startDate: today });
            handleClose();
          },
        },
        {
          group: "Task Actions",
          icon: <SomedayIcon />,
          label: "Schedule for Someday",
          shortcut: "⌘S",
          keywords: ["schedule", "someday", "task", "later"],
          action: () => {
            updateTask({ id: taskId, schedule: 2 });
            handleClose();
          },
        },
        {
          group: "Task Actions",
          icon: <CheckIcon />,
          label: "Complete Task",
          shortcut: "⇧⌘C",
          keywords: ["complete", "done", "finish", "task"],
          action: () => {
            completeTask(taskId);
            handleClose();
          },
        },
        {
          group: "Task Actions",
          icon: <TrashIcon />,
          label: "Delete Task",
          shortcut: "⌫",
          keywords: ["delete", "remove", "task"],
          action: () => {
            deleteTask(taskId);
            handleClose();
          },
        },
      );
    }

    // Create group
    commands.push(
      {
        group: "Create",
        icon: <PlusIcon />,
        label: "New Task",
        shortcut: "⌘N",
        keywords: ["new", "create", "add", "task"],
        action: () => {
          createTask("");
          handleClose();
        },
      },
      {
        group: "Create",
        icon: <PlusIcon />,
        label: "New Project",
        shortcut: "⌘⇧N",
        keywords: ["new", "create", "add", "project"],
        action: () => {
          createProject({ title: "New Project" });
          handleClose();
        },
      },
    );

    return commands;
  }, [selectedTaskId, handleClose]);

  const allCommands = buildCommands();
  const filteredCommands = allCommands.filter((cmd) =>
    matchesQuery(cmd, query),
  );

  // Group filtered commands
  const groups: { label: string; commands: Command[] }[] = [];
  for (const cmd of filteredCommands) {
    const existing = groups.find((g) => g.label === cmd.group);
    if (existing) {
      existing.commands.push(cmd);
    } else {
      groups.push({ label: cmd.group, commands: [cmd] });
    }
  }

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === "ArrowDown") {
        e.preventDefault();
        setActiveIndex((i) => Math.min(i + 1, filteredCommands.length - 1));
      } else if (e.key === "ArrowUp") {
        e.preventDefault();
        setActiveIndex((i) => Math.max(i - 1, 0));
      } else if (e.key === "Enter") {
        e.preventDefault();
        if (filteredCommands[activeIndex]) {
          filteredCommands[activeIndex].action();
        }
      } else if (e.key === "Escape") {
        e.preventDefault();
        handleClose();
      }
    },
    [filteredCommands, activeIndex, handleClose],
  );

  if (!showPalette) return null;

  // Calculate flat index for each command to determine active state
  let flatIndex = 0;

  return (
    <>
      <div className="cmd-backdrop open" onClick={handleClose} />
      <div className="cmd-palette open">
        <div className="cmd-input-wrap">
          <div className="cmd-input-icon">
            <svg viewBox="0 0 16 16">
              <circle cx="7" cy="7" r="4.5" />
              <line x1="10.2" y1="10.2" x2="14" y2="14" />
            </svg>
          </div>
          <input
            ref={inputRef}
            className="cmd-input"
            placeholder="Type a command or search..."
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            onKeyDown={handleKeyDown}
            role="combobox"
            aria-expanded="true"
            aria-controls="cmd-palette-results"
            aria-activedescendant={`cmd-item-${activeIndex}`}
            aria-label="Type a command or search"
          />
          <span className="cmd-shortcut">⌘K</span>
        </div>
        <div className="cmd-results" id="cmd-palette-results" role="listbox" aria-label="Commands">
          {groups.map((group) => (
            <div key={group.label} role="group" aria-label={group.label}>
              <div className="cmd-group-label" aria-hidden="true">{group.label}</div>
              {group.commands.map((cmd) => {
                const currentIndex = flatIndex++;
                const isActive = currentIndex === activeIndex;
                return (
                  <div
                    key={cmd.label}
                    id={`cmd-item-${currentIndex}`}
                    role="option"
                    aria-selected={isActive}
                    className={`cmd-item${isActive ? " active" : ""}`}
                    onMouseEnter={() => setActiveIndex(currentIndex)}
                    onClick={cmd.action}
                  >
                    <span className="cmd-item-icon" aria-hidden="true">{cmd.icon}</span>
                    <span className="cmd-item-label">{cmd.label}</span>
                    {cmd.shortcut && (
                      <span className="cmd-item-shortcut">{cmd.shortcut}</span>
                    )}
                  </div>
                );
              })}
            </div>
          ))}
          {filteredCommands.length === 0 && (
            <div className="cmd-group-label">No results</div>
          )}
        </div>
      </div>
    </>
  );
}
