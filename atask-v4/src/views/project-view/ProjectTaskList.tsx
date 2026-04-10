import { Fragment, useCallback, useEffect, useRef, useState } from 'react';
import useForeignDropIndex from '../../hooks/useForeignDropIndex';
import TaskRow, { shouldHandleTaskRowPointerDown } from '../../components/TaskRow';
import TaskInlineEditor from '../../components/TaskInlineEditor';
import NewTaskRow from '../../components/NewTaskRow';
import DropSlot from '../../components/task-row/DropSlot';
import DragOverlay from '../../components/DragOverlay';
import usePointerReorder from '../../hooks/usePointerReorder';
import type { ReorderMove, Task } from '../../types';
import { startTaskPointerDrag, endTaskPointerDrag, updateTask, reorderTasks, $tasks, $projects, $selectedTaskIds } from '../../store/index';
import { useStore } from '@nanostores/react';
import { $taskPointerDrag } from '../../store/ui';
import { todayLocal, tomorrowLocal } from '../../lib/dates';

interface ProjectTaskListProps {
  tasks: Task[];
  projectId: string;
  /**
   * Stable identifier for this task list instance. Each section's
   * ProjectTaskList must pass a different listId so cross-section drag
   * detection can tell them apart. Top-level (sectionless) lists pass
   * `task-sectionless:${projectId}`; per-section lists pass
   * `task-section:${section.id}`.
   */
  listId: string;
  expandedTaskId: string | null;
  selectedTaskId: string | null;
  selectedTaskIds: Set<string>;
  onSelectTask: (id: string) => void;
  onExpandTask: (id: string) => void;
  onCloseExpandedTask: () => void;
  onCreateTask: (title: string) => void;
  onReorderTasks: (moves: ReorderMove[]) => Promise<void>;
  onTaskDrop?: (taskId: string) => void;
}

export default function ProjectTaskList({
  tasks,
  projectId,
  listId,
  expandedTaskId,
  selectedTaskId,
  selectedTaskIds,
  onSelectTask,
  onExpandTask,
  onCloseExpandedTask,
  onCreateTask,
  onReorderTasks,
  onTaskDrop,
}: ProjectTaskListProps) {
  const [isDropTarget, setIsDropTarget] = useState(false);
  const taskDrag = useStore($taskPointerDrag);
  /**
   * Compute the insertion index for a cross-list drop by walking the
   * task elements inside `targetWrapper` and finding the one whose
   * vertical center is below the cursor. Mirrors the within-list
   * getDropIndex logic in usePointerReorder. Returns the index in the
   * TARGET list's order (0..N inclusive).
   */
  const computeTargetDropIndex = (targetWrapper: Element, cursorY: number): number => {
    const taskNodes = Array.from(
      targetWrapper.querySelectorAll<HTMLElement>('[data-task-id]'),
    );
    for (let i = 0; i < taskNodes.length; i += 1) {
      const rect = taskNodes[i].getBoundingClientRect();
      const centerY = rect.top + rect.height / 2;
      if (cursorY < centerY) return i;
    }
    return taskNodes.length;
  };

  /**
   * Apply a cross-list move to the task: update the section/sectionless
   * pointer AND reorder the destination list so the dropped task lands
   * at the position the user released over (matching the foreign drop
   * indicator). Without the reorder, the task lands at its old index
   * inside the new list, ignoring the insertion line — Codex P2.
   *
   * Skips the reorder when targetSectionId is unchanged (within-section
   * paths use the within-list splice in usePointerReorder instead).
   */
  const applyCrossSectionDrop = async (
    taskId: string,
    targetSectionId: string | null,
    targetTaskIds: string[],
    insertionIndex: number,
  ) => {
    // Move the task into the target section first.
    await updateTask({ id: taskId, sectionId: targetSectionId });

    // Build the new ordering for the target list. Filter out the
    // dragged task in case it was already there (cross-list path
    // shouldn't normally happen since the same-target guards return
    // false above, but keep this defensive).
    const filtered = targetTaskIds.filter((id) => id !== taskId);
    const clampedIndex = Math.max(0, Math.min(insertionIndex, filtered.length));
    const newOrder = [
      ...filtered.slice(0, clampedIndex),
      taskId,
      ...filtered.slice(clampedIndex),
    ];

    // Convert ordering to {id, index} moves. Only emit moves for tasks
    // whose index actually changes — avoid touching unrelated rows.
    const allTasks = $tasks.get();
    const moves: Array<{ id: string; index: number }> = [];
    newOrder.forEach((id, idx) => {
      const task = allTasks.find((t) => t.id === id);
      if (!task || task.index !== idx) {
        moves.push({ id, index: idx });
      }
    });
    if (moves.length > 0) {
      await reorderTasks(moves);
    }
  };

  const handleCrossListDrop = (
    taskId: string,
    target: Element,
    cursor: { x: number; y: number },
  ) => {
    const sidebarItemId = target.getAttribute('data-sidebar-item-id');
    const sidebarItemKind = target.getAttribute('data-sidebar-item-kind');

    if (sidebarItemKind === 'project' && sidebarItemId && sidebarItemId !== projectId) {
      const allProjects = $projects.get();
      const project = allProjects.find((p) => p.id === sidebarItemId);
      updateTask({ id: taskId, projectId: sidebarItemId, areaId: project?.areaId ?? null, schedule: 0, startDate: null, timeSlot: null });
      return true;
    }

    if (sidebarItemKind === 'area' && sidebarItemId) {
      updateTask({ id: taskId, areaId: sidebarItemId, projectId: null, schedule: 0, startDate: null, timeSlot: null });
      return true;
    }

    if (sidebarItemKind === 'section' && sidebarItemId) {
      // Same-section drops fall through to within-list reorder (return
      // false) so the normal drop-index splice runs. Only cross-section
      // moves actually update the sectionId.
      const draggedTask = $tasks.get().find((t) => t.id === taskId);
      if (draggedTask && draggedTask.sectionId === sidebarItemId) {
        return false;
      }
      // Compute target ordering: tasks already in this section, sorted
      // by their current index, then place the dragged task at the
      // position the cursor released over.
      const targetTaskIds = $tasks
        .get()
        .filter((t) => t.projectId === projectId && t.sectionId === sidebarItemId && t.status === 0)
        .sort((a, b) => a.index - b.index)
        .map((t) => t.id);
      const insertionIndex = computeTargetDropIndex(target, cursor.y);
      void applyCrossSectionDrop(taskId, sidebarItemId, targetTaskIds, insertionIndex);
      return true;
    }

    if (sidebarItemKind === 'sectionless') {
      // Same-target guard: if the dragged task is already sectionless,
      // fall through to within-list reorder instead of firing a no-op
      // updateTask that would skip the index splice.
      const draggedTask = $tasks.get().find((t) => t.id === taskId);
      if (draggedTask && draggedTask.sectionId == null) {
        return false;
      }
      const targetTaskIds = $tasks
        .get()
        .filter((t) => t.projectId === projectId && t.sectionId == null && t.status === 0)
        .sort((a, b) => a.index - b.index)
        .map((t) => t.id);
      const insertionIndex = computeTargetDropIndex(target, cursor.y);
      void applyCrossSectionDrop(taskId, null, targetTaskIds, insertionIndex);
      return true;
    }

    const closestNavItem = target.closest('[data-sidebar-item-kind="nav"]');
    if (closestNavItem) {
      const view = closestNavItem.getAttribute('data-sidebar-item-id');
      if (view === 'inbox') {
        updateTask({ id: taskId, schedule: 0, startDate: null, timeSlot: null, projectId: null, areaId: null, sectionId: null });
        return true;
      }
      if (view === 'today') {
        const today = todayLocal();
        updateTask({ id: taskId, schedule: 1, startDate: today, projectId: null, areaId: null, sectionId: null });
        return true;
      }
      if (view === 'upcoming') {
        updateTask({ id: taskId, schedule: 3, startDate: tomorrowLocal(), projectId: null, areaId: null, sectionId: null });
        return true;
      }
      if (view === 'someday') {
        updateTask({ id: taskId, schedule: 2, projectId: null, areaId: null, sectionId: null });
        return true;
      }
    }

    return false;
  };

  const containerRef = useRef<HTMLDivElement | null>(null);
  const { reorderState, getPointerHandlers, registerItem, getItemRect } = usePointerReorder({
    getSelectedIds: () => $selectedTaskIds.get(),
    items: tasks,
    listId,
    kind: 'task',
    onReorder: onReorderTasks,
    shouldHandlePointerDown: (event) => shouldHandleTaskRowPointerDown(event.target),
    onDragStart: startTaskPointerDrag,
    onDragEnd: endTaskPointerDrag,
    onCrossListDrop: handleCrossListDrop,
  });

  // Foreign drop indicator: show an insertion line when a task being
  // dragged from another section/list is hovering this list.
  const taskItemRefs = useRef<Map<string, HTMLDivElement>>(new Map());

  // Cache per-id ref callback. See SidebarProjectGroup for the rationale:
  // we re-render on every cursor move during a drag, and a fresh closure
  // per render would null + reset every TaskRow ref, breaking pointer
  // capture continuity. Stable identity per id keeps React from
  // re-attaching refs.
  const registerTaskItemCacheRef = useRef<Map<string, (node: HTMLDivElement | null) => void>>(
    new Map(),
  );
  useEffect(() => {
    registerTaskItemCacheRef.current = new Map();
  }, [registerItem]);
  const registerTaskItem = useCallback((id: string) => {
    let cached = registerTaskItemCacheRef.current.get(id);
    if (!cached) {
      const hookRegister = registerItem(id);
      cached = (node: HTMLDivElement | null) => {
        if (node) taskItemRefs.current.set(id, node);
        else taskItemRefs.current.delete(id);
        hookRegister(node);
      };
      registerTaskItemCacheRef.current.set(id, cached);
    }
    return cached;
  }, [registerItem]);

  const foreignDrop = useForeignDropIndex({
    listId,
    kind: 'task',
    containerRef,
    getItemElements: () => {
      const elements: HTMLDivElement[] = [];
      for (const task of tasks) {
        const el = taskItemRefs.current.get(task.id);
        if (el) elements.push(el);
      }
      return elements;
    },
  });
  const draggedTaskIndex = reorderState.activeId
    ? tasks.findIndex((task) => task.id === reorderState.activeId)
    : -1;
  const isDragging = reorderState.isPointerDragging;
  const itemWidth = reorderState.activeId ? getItemRect(reorderState.activeId)?.width ?? null : null;

  const handlePointerEnter = () => {
    if (taskDrag.activeTaskId && onTaskDrop) {
      setIsDropTarget(true);
    }
  };

  const handlePointerLeave = () => {
    setIsDropTarget(false);
  };

  const handlePointerUp = () => {
    if (taskDrag.activeTaskId && onTaskDrop) {
      onTaskDrop(taskDrag.activeTaskId);
    }
    setIsDropTarget(false);
  };

  const renderDropZone = (index: number) => {
    const localVisible =
      isDragging &&
      reorderState.dropIndex === index &&
      index !== draggedTaskIndex &&
      index !== draggedTaskIndex + 1;
    // Foreign-list drag landing in this list -> mirror its dropIndex.
    const foreignVisible =
      !isDragging &&
      foreignDrop.isForeignHovering &&
      foreignDrop.dropIndex === index;

    if (!localVisible && !foreignVisible) return null;

    const edgeClass = index === 0
      ? ' task-drop-zone-edge-top'
      : index === tasks.length
        ? ' task-drop-zone-edge-bottom'
        : '';

    return (
      <div
        key={`drop-zone-${index}`}
        className={`task-drop-zone${edgeClass}`}
      >
        <DropSlot />
      </div>
    );
  };

  const renderDragClone = (id: string) => {
    const task = tasks.find((t) => t.id === id);
    if (!task) return null;
    return (
      <div
        style={{
          background: 'var(--sidebar-hover)',
          borderRadius: 'var(--radius-md)',
          boxShadow: '0 4px 12px rgba(0,0,0,0.15)',
          padding: '8px 12px',
        }}
      >
        <span style={{ fontSize: 'var(--text-base)', color: 'var(--ink-primary)' }}>
          {task.title}
        </span>
      </div>
    );
  };

  // The "sectionless" data-sidebar-item-id is only set for the top-level
  // ProjectTaskList. Per-section instances must leave it off so that the
  // closest() walk from a task row inside lands on the section block's
  // outer wrapper, not on the inner sectionless wrapper.
  const isTopLevelSectionless = listId.startsWith('task-sectionless:');
  return (
    <div
      ref={containerRef}
      onPointerEnter={handlePointerEnter}
      onPointerLeave={handlePointerLeave}
      onPointerUp={handlePointerUp}
      data-sidebar-item-kind={isTopLevelSectionless ? 'sectionless' : undefined}
      data-sidebar-item-id={isTopLevelSectionless ? listId : undefined}
      style={isDropTarget ? { background: 'var(--accent-subtle)', borderRadius: 'var(--radius-md)' } : undefined}
    >
      {tasks.map((task, index) => (
        <Fragment key={task.id}>
          {renderDropZone(index)}
          {expandedTaskId === task.id ? (
            <TaskInlineEditor
              task={task}
              onClose={onCloseExpandedTask}
            />
          ) : (
            <TaskRow
              task={task}
              isSelected={selectedTaskId === task.id}
              isMultiSelected={selectedTaskIds.has(task.id)}
              taskList={tasks}
              hideProjectPill
              onClick={() => onSelectTask(task.id)}
              onDoubleClick={() => onExpandTask(task.id)}
              reorderRef={registerTaskItem(task.id)}
              reorderHandlers={getPointerHandlers(task.id)}
              isReordering={reorderState.activeId === task.id}
            />
          )}
        </Fragment>
      ))}
      {renderDropZone(tasks.length)}
      <NewTaskRow onCreate={onCreateTask} />
      <DragOverlay
        activeId={reorderState.activeId}
        cursorX={reorderState.cursorX}
        cursorY={reorderState.cursorY}
        itemWidth={itemWidth}
        renderClone={renderDragClone}
      />
    </div>
  );
}
