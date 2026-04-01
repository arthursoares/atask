import { useState, useEffect, useRef, useCallback, useMemo } from 'react';
import { useStore } from '@nanostores/react';
import {
  $tasks,
  $projects,
  $showSearch,
  setActiveView,
  selectTask,
} from '../store/index';
import type { Task, Project } from '../types';

interface SearchResult {
  type: "task" | "project";
  id: string;
  title: string;
  projectName: string | null;
  snippet: string | null;
  task?: Task;
  project?: Project;
}

function fuzzyMatch(query: string, haystack: string): boolean {
  const words = query.toLowerCase().split(/\s+/).filter(Boolean);
  const lower = haystack.toLowerCase();
  return words.every((w) => lower.includes(w));
}

function extractSnippet(notes: string, query: string): string | null {
  if (!notes) return null;
  const lower = notes.toLowerCase();
  const words = query.toLowerCase().split(/\s+/).filter(Boolean);
  // Find the first word that appears in notes
  for (const word of words) {
    const idx = lower.indexOf(word);
    if (idx >= 0) {
      const start = Math.max(0, idx - 30);
      const end = Math.min(notes.length, idx + word.length + 50);
      let snippet = notes.slice(start, end).replace(/\n/g, " ");
      if (start > 0) snippet = "..." + snippet;
      if (end < notes.length) snippet = snippet + "...";
      return snippet;
    }
  }
  return null;
}

const MAX_RESULTS = 20;

export default function SearchOverlay() {
  const showSearch = useStore($showSearch);
  const tasks = useStore($tasks);
  const projects = useStore($projects);

  const [query, setQuery] = useState("");
  const [activeIndex, setActiveIndex] = useState(0);
  const inputRef = useRef<HTMLInputElement>(null);
  const resultsRef = useRef<HTMLDivElement>(null);

  // Build a project lookup map
  const projectMap = useMemo(() => {
    const map = new Map<string, Project>();
    for (const p of projects) {
      map.set(p.id, p);
    }
    return map;
  }, [projects]);

  // Compute search results
  const results = useMemo((): SearchResult[] => {
    if (!query.trim()) return [];
    const out: SearchResult[] = [];

    // Search tasks
    for (const task of tasks) {
      if (out.length >= MAX_RESULTS) break;
      const haystack = [task.title, task.notes ?? ""].join(" ");
      if (fuzzyMatch(query, haystack)) {
        const proj = task.projectId ? projectMap.get(task.projectId) : null;
        const titleMatch = fuzzyMatch(query, task.title);
        const snippet = titleMatch ? null : extractSnippet(task.notes ?? "", query);
        out.push({
          type: "task",
          id: task.id,
          title: task.title || "Untitled task",
          projectName: proj?.title ?? null,
          snippet,
          task,
        });
      }
    }

    // Search projects
    for (const project of projects) {
      if (out.length >= MAX_RESULTS) break;
      if (fuzzyMatch(query, project.title)) {
        out.push({
          type: "project",
          id: project.id,
          title: project.title || "Untitled project",
          projectName: null,
          snippet: null,
          project,
        });
      }
    }

    return out;
  }, [query, tasks, projects, projectMap]);

  // Reset state when overlay opens
  useEffect(() => {
    if (showSearch) {
      setQuery("");
      setActiveIndex(0);
      setTimeout(() => {
        inputRef.current?.focus();
      }, 0);
    }
  }, [showSearch]);

  // Reset active index when query changes
  useEffect(() => {
    setActiveIndex(0);
  }, [query]);

  // Scroll active item into view
  useEffect(() => {
    if (!resultsRef.current) return;
    const active = resultsRef.current.querySelector(".cmd-item.active");
    if (active) {
      active.scrollIntoView({ block: "nearest" });
    }
  }, [activeIndex]);

  const handleClose = useCallback(() => {
    $showSearch.set(false);
    setQuery("");
  }, []);

  const handleSelect = useCallback(
    (result: SearchResult) => {
      if (result.type === 'project' && result.project) {
        setActiveView(`project-${result.project.id}`);
      } else if (result.type === 'task' && result.task) {
        const task = result.task;
        if (task.projectId) {
          setActiveView(`project-${task.projectId}`);
        } else if (task.schedule === 0) {
          setActiveView('inbox');
        } else if (task.schedule === 1) {
          setActiveView('today');
        } else if (task.schedule === 2) {
          setActiveView('someday');
        }
        selectTask(task.id);
      }
      handleClose();
    },
    [handleClose],
  );

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === "ArrowDown") {
        e.preventDefault();
        setActiveIndex((i) => Math.min(i + 1, results.length - 1));
      } else if (e.key === "ArrowUp") {
        e.preventDefault();
        setActiveIndex((i) => Math.max(i - 1, 0));
      } else if (e.key === "Enter") {
        e.preventDefault();
        if (results[activeIndex]) {
          handleSelect(results[activeIndex]);
        }
      } else if (e.key === "Escape") {
        e.preventDefault();
        handleClose();
      }
    },
    [results, activeIndex, handleSelect, handleClose],
  );

  if (!showSearch) return null;

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
            placeholder="Search tasks..."
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            onKeyDown={handleKeyDown}
          />
          <span className="cmd-shortcut">⌘F</span>
        </div>
        <div className="cmd-results" ref={resultsRef}>
          {results.map((result, index) => {
            const isActive = index === activeIndex;
            return (
              <div
                key={`${result.type}-${result.id}`}
                className={`cmd-item${isActive ? " active" : ""}`}
                onMouseEnter={() => setActiveIndex(index)}
                onClick={() => handleSelect(result)}
              >
                <span className="cmd-item-icon">
                  {result.type === "project" ? "📁" : "☐"}
                </span>
                <span className="cmd-item-label">
                  {result.title}
                  {result.projectName && (
                    <span style={{ opacity: 0.5, marginLeft: 8, fontSize: "0.85em" }}>
                      {result.projectName}
                    </span>
                  )}
                  {result.snippet && (
                    <span
                      style={{
                        display: "block",
                        opacity: 0.4,
                        fontSize: "0.8em",
                        marginTop: 2,
                        whiteSpace: "nowrap",
                        overflow: "hidden",
                        textOverflow: "ellipsis",
                      }}
                    >
                      {result.snippet}
                    </span>
                  )}
                </span>
                <span className="cmd-item-shortcut" style={{ textTransform: "capitalize" }}>
                  {result.type}
                </span>
              </div>
            );
          })}
          {query.trim() && results.length === 0 && (
            <div className="cmd-group-label">No results</div>
          )}
          {!query.trim() && (
            <div className="cmd-group-label">Type to search tasks and projects</div>
          )}
        </div>
      </div>
    </>
  );
}
