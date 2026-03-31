import { useState, useEffect, useRef, useCallback, useMemo } from "react";
import { useStore } from "@nanostores/react";
import {
  $showQuickMove,
  $selectedTaskId,
  useActiveProjects,
  useActiveAreas,
  updateTask,
} from "../store/index";
import type { Project, Area } from "../types";

interface ProjectOption {
  id: string | null;
  title: string;
  color: string;
  areaLabel: string | null;
}

export default function QuickMovePicker() {
  const showQuickMove = useStore($showQuickMove);
  const selectedTaskId = useStore($selectedTaskId);
  const setShowQuickMove = (v: boolean) => $showQuickMove.set(v);

  const projects = useActiveProjects();
  const areas = useActiveAreas();

  const [query, setQuery] = useState("");
  const [activeIndex, setActiveIndex] = useState(0);
  const inputRef = useRef<HTMLInputElement>(null);

  // Build the options list: "No Project (Inbox)" first, then projects grouped by area
  const options = useMemo(() => {
    const areaMap = new Map<string, Area>();
    for (const a of areas) areaMap.set(a.id, a);

    const result: ProjectOption[] = [
      { id: null, title: "No Project (Inbox)", color: "", areaLabel: null },
    ];

    // Projects with an area, grouped
    const withArea: { area: Area; projects: Project[] }[] = [];
    const noArea: Project[] = [];

    for (const p of projects) {
      if (p.areaId) {
        const area = areaMap.get(p.areaId);
        if (area) {
          const group = withArea.find((g) => g.area.id === area.id);
          if (group) {
            group.projects.push(p);
          } else {
            withArea.push({ area, projects: [p] });
          }
        } else {
          noArea.push(p);
        }
      } else {
        noArea.push(p);
      }
    }

    // Sort area groups by area index
    withArea.sort((a, b) => a.area.index - b.area.index);

    // Add ungrouped projects first
    for (const p of noArea) {
      result.push({
        id: p.id,
        title: p.title,
        color: p.color,
        areaLabel: null,
      });
    }

    // Add area-grouped projects
    for (const group of withArea) {
      for (const p of group.projects) {
        result.push({
          id: p.id,
          title: p.title,
          color: p.color,
          areaLabel: group.area.title,
        });
      }
    }

    return result;
  }, [projects, areas]);

  // Filter by query
  const filtered = useMemo(() => {
    if (!query) return options;
    const words = query.toLowerCase().split(/\s+/).filter(Boolean);
    return options.filter((opt) => {
      const haystack = [opt.title, opt.areaLabel ?? ""].join(" ").toLowerCase();
      return words.every((w) => haystack.includes(w));
    });
  }, [options, query]);

  // Reset state when picker opens
  useEffect(() => {
    if (showQuickMove) {
      setQuery("");
      setActiveIndex(0);
      setTimeout(() => inputRef.current?.focus(), 0);
    }
  }, [showQuickMove]);

  // Reset active index when query changes
  useEffect(() => {
    setActiveIndex(0);
  }, [query]);

  const handleClose = useCallback(() => {
    setShowQuickMove(false);
    setQuery("");
  }, [setShowQuickMove]);

  const handleSelect = useCallback(
    (option: ProjectOption) => {
      if (!selectedTaskId) return;
      updateTask({ id: selectedTaskId, projectId: option.id });
      handleClose();
    },
    [selectedTaskId, updateTask, handleClose],
  );

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === "ArrowDown") {
        e.preventDefault();
        setActiveIndex((i) => Math.min(i + 1, filtered.length - 1));
      } else if (e.key === "ArrowUp") {
        e.preventDefault();
        setActiveIndex((i) => Math.max(i - 1, 0));
      } else if (e.key === "Enter") {
        e.preventDefault();
        if (filtered[activeIndex]) {
          handleSelect(filtered[activeIndex]);
        }
      } else if (e.key === "Escape") {
        e.preventDefault();
        handleClose();
      }
    },
    [filtered, activeIndex, handleSelect, handleClose],
  );

  if (!showQuickMove) return null;

  // Group filtered options by area for display
  let currentArea: string | null | undefined = undefined;

  return (
    <>
      <div className="cmd-backdrop open" onClick={handleClose} />
      <div className="cmd-palette open quick-move">
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
            placeholder="Move to project..."
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            onKeyDown={handleKeyDown}
          />
          <span className="cmd-shortcut">⇧⌘M</span>
        </div>
        <div className="cmd-results">
          {filtered.map((opt, idx) => {
            // Show area header when the area changes
            let areaHeader: React.ReactNode = null;
            if (opt.areaLabel !== currentArea) {
              currentArea = opt.areaLabel;
              if (opt.areaLabel) {
                areaHeader = (
                  <div className="cmd-group-label">{opt.areaLabel}</div>
                );
              }
            }

            const isActive = idx === activeIndex;
            return (
              <div key={opt.id ?? "__inbox"}>
                {areaHeader}
                <div
                  className={`cmd-item${isActive ? " active" : ""}`}
                  onMouseEnter={() => setActiveIndex(idx)}
                  onClick={() => handleSelect(opt)}
                >
                  {opt.id ? (
                    <span
                      className="cmd-item-icon"
                      style={{
                        display: "inline-block",
                        width: 10,
                        height: 10,
                        borderRadius: "50%",
                        backgroundColor: opt.color || "var(--accent)",
                        flexShrink: 0,
                      }}
                    />
                  ) : (
                    <span className="cmd-item-icon">{"📥"}</span>
                  )}
                  <span className="cmd-item-label">{opt.title}</span>
                </div>
              </div>
            );
          })}
          {filtered.length === 0 && (
            <div className="cmd-group-label">No matching projects</div>
          )}
        </div>
      </div>
    </>
  );
}
