import { useEffect } from 'react';
import { useStore } from '@nanostores/react';
import { todayLocal } from '../lib/dates';
import {
  $activeView, $selectedTaskId, $selectedTaskIds, $expandedTaskId,
  $showPalette, $showQuickMove, $showSearch, $showSidebar, $showShortcuts, $tasks,
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
      if (meta && !shift && key === '1') { e.preventDefault(); $activeView.set('inbox'); return; }
      if (meta && !shift && key === '2') { e.preventDefault(); $activeView.set('today'); return; }
      if (meta && !shift && key === '3') { e.preventDefault(); $activeView.set('upcoming'); return; }
      if (meta && !shift && key === '4') { e.preventDefault(); $activeView.set('someday'); return; }
      if (meta && !shift && key === '5') { e.preventDefault(); $activeView.set('logbook'); return; }

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
        if (expandedTaskId) { $expandedTaskId.set(null); return; }
        if (selectedTaskId) { $selectedTaskId.set(null); return; }
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

      // Space: Create new task below selection (contextual ⌘N)
      if (key === ' ' && !meta && !shift) {
        e.preventDefault();
        createTask('');
        return;
      }

      // ⌘A: Select all tasks in current view
      if (meta && !shift && key === 'a') {
        e.preventDefault();
        const filteredTasks = getViewTasks();
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
        $activeView.set('settings');
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
      const today = todayLocal();
      const view = $activeView.get();
      const tasks = $tasks.get();

      let filtered: typeof tasks;
      if (view === 'inbox') {
        filtered = tasks.filter(t => t.schedule === 0 && t.status === 0);
      } else if (view === 'today') {
        filtered = tasks.filter(t => t.schedule === 1 && t.status === 0 && (t.todayIndex != null || (t.startDate && t.startDate.slice(0, 10) <= today)));
      } else if (view === 'someday') {
        filtered = tasks.filter(t => t.schedule === 2 && t.status === 0);
      } else if (view === 'logbook') {
        filtered = tasks.filter(t => t.status === 1 || t.status === 2);
      } else if (view.startsWith('project-')) {
        const projectId = view.slice('project-'.length);
        filtered = tasks.filter(t => t.projectId === projectId && t.status === 0);
      } else {
        filtered = tasks.filter(t => t.status === 0);
      }

      filtered.sort((a, b) => a.index - b.index);
      return filtered;
    }

    function navigateTasks(direction: number) {
      const tasks = getViewTasks();
      if (tasks.length === 0) return;

      const currentId = $selectedTaskId.get();
      if (!currentId) {
        $selectedTaskId.set(tasks[0].id);
        return;
      }

      const currentIndex = tasks.findIndex(t => t.id === currentId);
      const nextIndex = Math.max(0, Math.min(tasks.length - 1, currentIndex + direction));
      $selectedTaskId.set(tasks[nextIndex].id);
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
      $selectedTaskId.set(nextId);
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
