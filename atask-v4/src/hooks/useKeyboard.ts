import { useEffect } from 'react';
import { useStore } from '@nanostores/react';
import { todayLocal } from '../lib/dates';
import {
  $activeView, $selectedTaskId, $selectedTaskIds, $expandedTaskId,
  $showPalette, $showQuickMove, $showSearch, $showSidebar, $showShortcuts, $tasks,
  $activeTagFilters, $tagsByTaskId,
  $inbox, $today, $upcoming, $someday, $logbook,
  setActiveView, selectTask, clearSelectedTask, closeTaskEditor,
} from '../store';
import {
  createTask, completeTask, deleteTask, duplicateTask, updateTask,
  reorderTasks,
} from '../store';

function isEditingText(): boolean {
  const el = document.activeElement;
  if (!el) return false;
  const tag = el.tagName;
  if (tag === 'INPUT' || tag === 'TEXTAREA') return true;
  if ((el as HTMLElement).contentEditable === 'true') return true;
  return false;
}

export default function useKeyboard() {
  const showPalette = useStore($showPalette);
  const expandedTaskId = useStore($expandedTaskId);

  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      const meta = e.metaKey || e.ctrlKey;
      const shift = e.shiftKey;
      const key = e.key;

      // Read current state imperatively
      const selectedTaskId = $selectedTaskId.get();
      const showSidebar = $showSidebar.get();

      // --- Always-active shortcuts (work even in text fields) ---

      // ⌘1-5: Navigate views
      if (meta && !shift && key === '1') { e.preventDefault(); setActiveView('inbox'); return; }
      if (meta && !shift && key === '2') { e.preventDefault(); setActiveView('today'); return; }
      if (meta && !shift && key === '3') { e.preventDefault(); setActiveView('upcoming'); return; }
      if (meta && !shift && key === '4') { e.preventDefault(); setActiveView('someday'); return; }
      if (meta && !shift && key === '5') { e.preventDefault(); setActiveView('logbook'); return; }

      // ⇧⌘O: Toggle command palette (Things-compatible — NOT ⌘K which is Complete)
      if (meta && shift && (key === 'o' || key === 'O')) {
        e.preventDefault();
        $showPalette.set(!showPalette);
        return;
      }

      // ⌘F: Open search
      if (meta && !shift && key === 'f') {
        e.preventDefault();
        $showSearch.set(!$showSearch.get());
        return;
      }

      // ⌘?: Show shortcuts help
      if (meta && shift && key === '/') {
        e.preventDefault();
        $showShortcuts.set(!$showShortcuts.get());
        return;
      }

      // ⌘/: Toggle sidebar
      if (meta && !shift && key === '/') {
        e.preventDefault();
        $showSidebar.set(!showSidebar);
        return;
      }

      // --- Text-field-blocked shortcuts (skip if user is typing) ---
      if (isEditingText()) return;

      // ⌘N: New task — click the NewTaskRow to enter editing mode with focus
      if (meta && !shift && key === 'n') {
        e.preventDefault();
        const newTaskRow = document.querySelector('.new-task-inline');
        if (newTaskRow) {
          (newTaskRow as HTMLElement).click();
        } else {
          createTask('');
        }
        return;
      }

      // Escape: Deselect task / close expanded
      if (key === 'Escape') {
        if (expandedTaskId) { closeTaskEditor(); return; }
        if (selectedTaskId) { clearSelectedTask(); return; }
        return;
      }

      // ⌘K: Open command palette (alternative to ⇧⌘O)
      if (meta && !shift && key === 'k') {
        e.preventDefault();
        $showPalette.set(!showPalette);
        return;
      }

      // ⇧⌘P: Open command palette (VS Code convention)
      if (meta && shift && (key === 'p' || key === 'P')) {
        e.preventDefault();
        $showPalette.set(!showPalette);
        return;
      }

      // ⇧⌘C: Complete selected task
      if (meta && shift && (key === 'c' || key === 'C')) {
        e.preventDefault();
        if (selectedTaskId) completeTask(selectedTaskId);
        return;
      }

      // Backspace/Delete: Delete selected task
      if (key === 'Backspace' || key === 'Delete') {
        if (selectedTaskId) {
          e.preventDefault();
          deleteTask(selectedTaskId);
        }
        return;
      }

      // ⌘T: Schedule selected task for Today
      if (meta && !shift && key === 't') {
        e.preventDefault();
        if (selectedTaskId) {
          const today = todayLocal();
          updateTask({ id: selectedTaskId, schedule: 1, startDate: today });
        }
        return;
      }

      // ⌘E: Schedule selected task for This Evening
      if (meta && !shift && key === 'e') {
        e.preventDefault();
        if (selectedTaskId) {
          const today = todayLocal();
          updateTask({ id: selectedTaskId, schedule: 1, timeSlot: 'evening', startDate: today });
        }
        return;
      }

      // ⌘O: Schedule selected task for Someday (without shift)
      if (meta && !shift && (key === 'o')) {
        e.preventDefault();
        if (selectedTaskId) updateTask({ id: selectedTaskId, schedule: 2 });
        return;
      }

      // Enter: Open detail panel for selected task
      if (key === 'Enter' && !meta && !shift) {
        if (selectedTaskId && !expandedTaskId) {
          e.preventDefault();
          // selectedTaskId already shows the detail panel
        }
        return;
      }

      // ⌘D: Duplicate selected task
      if (meta && !shift && key === 'd') {
        e.preventDefault();
        if (selectedTaskId) {
          duplicateTask(selectedTaskId);
        }
        return;
      }

      // Space: Create a new task below the currently-selected task.
      //
      // Gated on having a task actually selected so Space retains its
      // universal "activate focused button" semantics everywhere else in
      // the app (sidebar nav items, toolbar buttons, checkboxes, etc.).
      // Without the gate, tabbing to any interactive element and pressing
      // Space would silently create a ghost empty task instead of firing
      // the button.
      if (key === ' ' && !meta && !shift && selectedTaskId) {
        e.preventDefault();
        createTask('');
        return;
      }

      // ⌘A: Select all tasks in current view
      if (meta && !shift && key === 'a') {
        e.preventDefault();
        const filteredTasks = getViewTasks();
        clearSelectedTask();
        $selectedTaskIds.set(new Set(filteredTasks.map(t => t.id)));
        return;
      }

      // ⇧⌘M: Open QuickMovePicker
      if (meta && shift && (key === 'm' || key === 'M')) {
        e.preventDefault();
        $showQuickMove.set(true);
        return;
      }

      // ⌘,: Navigate to Settings
      if (meta && !shift && key === ',') {
        e.preventDefault();
        setActiveView('settings');
        return;
      }

      // ⌘↑: Move selected task up in the list
      if (meta && !shift && key === 'ArrowUp') {
        e.preventDefault();
        moveTask(-1);
        return;
      }

      // ⌘↓: Move selected task down in the list
      if (meta && !shift && key === 'ArrowDown') {
        e.preventDefault();
        moveTask(1);
        return;
      }

      // ⇧↑/⇧↓: Extend selection
      if (shift && !meta && (key === 'ArrowUp' || key === 'ArrowDown')) {
        e.preventDefault();
        extendSelection(key === 'ArrowDown' ? 1 : -1);
        return;
      }

      // Arrow keys: Navigate task list (j/k vim-style also)
      if (key === 'ArrowDown' || key === 'j') {
        e.preventDefault();
        navigateTasks(1);
        return;
      }
      if (key === 'ArrowUp' || key === 'k') {
        e.preventDefault();
        navigateTasks(-1);
        return;
      }
    };

    function getViewTasks() {
      // Read directly from the computed selector atoms so keyboard arrow
      // navigation cycles through exactly what the view renders — including
      // tag filters, completed-today inclusion, and the correct ordering.
      // Previously this function re-implemented the filters and drifted:
      // under an active tag filter it would iterate over "hidden" tasks the
      // view was hiding.
      const view = $activeView.get();
      if (view === 'inbox') return $inbox.get();
      if (view === 'today') return $today.get();
      if (view === 'someday') return $someday.get();
      if (view === 'logbook') return $logbook.get();
      if (view === 'upcoming') {
        // $upcoming returns grouped date buckets; flatten to a single list
        // in render order for arrow navigation.
        return $upcoming.get().flatMap((group) => group.tasks);
      }

      // Project / area views: replicate the same tag-filtered, status-
      // aware filter inline since those are exposed as hooks, not atoms.
      const tasks = $tasks.get();
      const filters = $activeTagFilters.get();
      const tagMap = $tagsByTaskId.get();
      const passesTagFilter = (taskId: string): boolean => {
        if (filters.size === 0) return true;
        const taskTagIds = tagMap.get(taskId);
        if (!taskTagIds) return false;
        for (const tagId of filters) {
          if (taskTagIds.has(tagId)) return true;
        }
        return false;
      };

      if (view.startsWith('project-')) {
        const projectId = view.slice('project-'.length);
        return tasks
          .filter((t) => t.projectId === projectId && t.status === 0 && passesTagFilter(t.id))
          .sort((a, b) => a.index - b.index);
      }

      if (view.startsWith('area-')) {
        const areaId = view.slice('area-'.length);
        return tasks
          .filter((t) => t.areaId === areaId && t.projectId == null && t.status === 0 && passesTagFilter(t.id))
          .sort((a, b) => a.index - b.index);
      }

      // Fallback: all active tasks.
      return tasks.filter((t) => t.status === 0 && passesTagFilter(t.id)).sort((a, b) => a.index - b.index);
    }

    function navigateTasks(direction: number) {
      const tasks = getViewTasks();
      if (tasks.length === 0) return;

      const currentId = $selectedTaskId.get();
      let nextId: string;
      if (!currentId) {
        nextId = tasks[0].id;
      } else {
        const currentIndex = tasks.findIndex(t => t.id === currentId);
        if (currentIndex < 0) {
          // Current selection is outside the visible list (view changed,
          // selection points at a task that no longer passes the filter).
          // Jump to the first/last visible task by direction.
          nextId = direction > 0 ? tasks[0].id : tasks[tasks.length - 1].id;
        } else {
          const nextIndex = Math.max(0, Math.min(tasks.length - 1, currentIndex + direction));
          nextId = tasks[nextIndex].id;
        }
      }
      selectTask(nextId);

      // Scroll the newly-selected task into view so keyboard navigation
      // follows the eye. Uses a requestAnimationFrame because the DOM
      // isn't guaranteed to have applied the `selected` class by the
      // time selectTask() returns (nanostores schedules updates).
      requestAnimationFrame(() => {
        const row = document.querySelector(`[data-task-id="${nextId}"]`);
        if (row instanceof HTMLElement) {
          row.scrollIntoView({ block: 'nearest' });
        }
      });
    }

    function extendSelection(direction: number) {
      const tasks = getViewTasks();
      if (tasks.length === 0) return;

      const currentId = $selectedTaskId.get();
      if (!currentId) return;

      const currentIndex = tasks.findIndex(t => t.id === currentId);
      const nextIndex = Math.max(0, Math.min(tasks.length - 1, currentIndex + direction));
      if (nextIndex === currentIndex) return;

      const nextId = tasks[nextIndex].id;
      const next = new Set($selectedTaskIds.get());
      next.add(currentId);
      next.add(nextId);
      $selectedTaskIds.set(next);
      selectTask(nextId, { preserveMultiSelection: true });
    }

    function moveTask(direction: number) {
      const currentId = $selectedTaskId.get();
      if (!currentId) return;

      const tasks = getViewTasks();
      const currentIndex = tasks.findIndex(t => t.id === currentId);
      if (currentIndex < 0) return;

      const swapIndex = currentIndex + direction;
      if (swapIndex < 0 || swapIndex >= tasks.length) return;

      const current = tasks[currentIndex];
      const swap = tasks[swapIndex];

      // Swap indices between the two tasks
      reorderTasks([
        { id: current.id, index: swap.index },
        { id: swap.id, index: current.index },
      ]);
    }

    document.addEventListener('keydown', handler);
    return () => document.removeEventListener('keydown', handler);
  }, [showPalette, expandedTaskId]);
}
