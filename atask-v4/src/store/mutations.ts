import * as tauri from '../hooks/useTauri';

// Sync hook — set by useSync to avoid circular imports.
// Called after every mutation so sync can flush + pull.
let onMutation: (() => void) | null = null;
export function setOnMutation(cb: () => void) { onMutation = cb; }

function notifySync() { onMutation?.(); }
import { $tasks, $taskTags } from './tasks';
import { $projects } from './projects';
import { $areas } from './areas';
import { $sections } from './sections';
import { $tags } from './tags';
import { $checklistItems } from './checklist';
import {
  $activeView,
  $selectedTaskId,
  $expandedTaskId,
  $activeTagFilters,
} from './ui';
import type {
  Task,
  Project,
  Area,
  Section,
  Tag,
  ChecklistItem,
  CreateTaskParams,
  UpdateTaskParams,
  CreateProjectParams,
  UpdateProjectParams,
  CreateAreaParams,
  UpdateAreaParams,
  CreateSectionParams,
  UpdateSectionParams,
  CreateTagParams,
  UpdateTagParams,
  CreateChecklistItemParams,
  UpdateChecklistItemParams,
  ReorderMove,
} from '../types';

// --- Helpers ---

function replaceItem<T extends { id: string }>(arr: T[], item: T): T[] {
  return arr.map((x) => (x.id === item.id ? item : x));
}

function removeItem<T extends { id: string }>(arr: T[], id: string): T[] {
  return arr.filter((x) => x.id !== id);
}

// --- Data loading ---

export async function loadAll(): Promise<void> {
  const data = await tauri.loadAll();
  $tasks.set(data.tasks);
  $projects.set(data.projects);
  $areas.set(data.areas);
  $sections.set(data.sections);
  $tags.set(data.tags);
  $taskTags.set(data.taskTags);
  $checklistItems.set(data.checklistItems);
}

// --- Task actions ---

export async function createTask(title: string): Promise<Task> {
  const view = $activeView.get();
  const params: CreateTaskParams = { title };

  if (view === 'today') {
    params.schedule = 1;
    const hour = new Date().getHours();
    params.timeSlot = hour < 17 ? 'morning' : 'evening';
  } else if (view === 'someday') {
    params.schedule = 2;
  } else if (view.startsWith('project-')) {
    params.projectId = view.slice('project-'.length);
  }

  const task = await tauri.createTask(params);
  $tasks.set([...$tasks.get(), task]);
  notifySync();
  return task;
}

export async function completeTask(id: string): Promise<void> {
  const task = await tauri.completeTask(id);
  $tasks.set(replaceItem($tasks.get(), task));
  notifySync();
}

export async function cancelTask(id: string): Promise<void> {
  const task = await tauri.cancelTask(id);
  $tasks.set(replaceItem($tasks.get(), task));
  notifySync();
}

export async function reopenTask(id: string): Promise<void> {
  const task = await tauri.reopenTask(id);
  $tasks.set(replaceItem($tasks.get(), task));
  notifySync();
}

export async function updateTask(params: UpdateTaskParams): Promise<void> {
  const task = await tauri.updateTask(params);
  $tasks.set(replaceItem($tasks.get(), task));
  notifySync();
}

export async function duplicateTask(id: string): Promise<void> {
  const task = await tauri.duplicateTask(id);
  $tasks.set([...$tasks.get(), task]);
  notifySync();
}

export async function deleteTask(id: string): Promise<void> {
  await tauri.deleteTask(id);
  $tasks.set(removeItem($tasks.get(), id));
  $checklistItems.set($checklistItems.get().filter((ci) => ci.taskId !== id));
  $taskTags.set($taskTags.get().filter((tt) => tt.taskId !== id));
  if ($selectedTaskId.get() === id) $selectedTaskId.set(null);
  if ($expandedTaskId.get() === id) $expandedTaskId.set(null);
  notifySync();
}

export async function reorderTasks(moves: ReorderMove[]): Promise<void> {
  await tauri.reorderTasks(moves);
  const indexed = new Map(moves.map((m) => [m.id, m.index]));
  $tasks.set(
    $tasks.get().map((t) => {
      const idx = indexed.get(t.id);
      return idx !== undefined ? { ...t, index: idx } : t;
    }),
  );
}

export async function setTodayIndex(id: string, index: number): Promise<void> {
  const task = await tauri.setTodayIndex(id, index);
  $tasks.set(replaceItem($tasks.get(), task));
}

export async function moveTaskToSection(id: string, sectionId: string | null): Promise<void> {
  const task = await tauri.moveTaskToSection(id, sectionId);
  $tasks.set(replaceItem($tasks.get(), task));
}

// --- Project actions ---

export async function createProject(params: CreateProjectParams): Promise<Project> {
  const project = await tauri.createProject(params);
  $projects.set([...$projects.get(), project]);
  notifySync();
  return project;
}

export async function updateProject(params: UpdateProjectParams): Promise<void> {
  const project = await tauri.updateProject(params);
  $projects.set(replaceItem($projects.get(), project));
}

export async function completeProject(id: string): Promise<void> {
  const project = await tauri.completeProject(id);
  $projects.set(replaceItem($projects.get(), project));
}

export async function reopenProject(id: string): Promise<void> {
  const project = await tauri.reopenProject(id);
  $projects.set(replaceItem($projects.get(), project));
}

export async function deleteProject(id: string): Promise<void> {
  await tauri.deleteProject(id);
  const taskIds = new Set(
    $tasks.get().filter((t) => t.projectId === id).map((t) => t.id),
  );
  $projects.set(removeItem($projects.get(), id));
  $tasks.set($tasks.get().filter((t) => t.projectId !== id));
  $sections.set($sections.get().filter((s) => s.projectId !== id));
  $checklistItems.set($checklistItems.get().filter((ci) => !taskIds.has(ci.taskId)));
  $taskTags.set($taskTags.get().filter((tt) => !taskIds.has(tt.taskId)));
  if ($activeView.get() === `project-${id}`) $activeView.set('inbox');
  if (taskIds.has($selectedTaskId.get() ?? '')) $selectedTaskId.set(null);
  if (taskIds.has($expandedTaskId.get() ?? '')) $expandedTaskId.set(null);
  notifySync();
}

export async function moveProjectToArea(id: string, areaId: string | null): Promise<void> {
  const project = await tauri.moveProjectToArea(id, areaId);
  $projects.set(replaceItem($projects.get(), project));
}

export async function reorderProjects(moves: ReorderMove[]): Promise<void> {
  await tauri.reorderProjects(moves);
  const indexed = new Map(moves.map((m) => [m.id, m.index]));
  $projects.set(
    $projects.get().map((p) => {
      const idx = indexed.get(p.id);
      return idx !== undefined ? { ...p, index: idx } : p;
    }),
  );
}

// --- Area actions ---

export async function createArea(params: CreateAreaParams): Promise<Area> {
  const area = await tauri.createArea(params);
  $areas.set([...$areas.get(), area]);
  return area;
}

export async function updateArea(params: UpdateAreaParams): Promise<void> {
  const area = await tauri.updateArea(params);
  $areas.set(replaceItem($areas.get(), area));
}

export async function deleteArea(id: string): Promise<void> {
  await tauri.deleteArea(id);
  $areas.set(removeItem($areas.get(), id));
}

export async function toggleAreaArchived(id: string): Promise<void> {
  const area = await tauri.toggleAreaArchived(id);
  $areas.set(replaceItem($areas.get(), area));
}

export async function reorderAreas(moves: ReorderMove[]): Promise<void> {
  await tauri.reorderAreas(moves);
  const indexed = new Map(moves.map((m) => [m.id, m.index]));
  $areas.set(
    $areas.get().map((a) => {
      const idx = indexed.get(a.id);
      return idx !== undefined ? { ...a, index: idx } : a;
    }),
  );
}

// --- Section actions ---

export async function createSection(params: CreateSectionParams): Promise<Section> {
  const section = await tauri.createSection(params);
  $sections.set([...$sections.get(), section]);
  return section;
}

export async function updateSection(params: UpdateSectionParams): Promise<void> {
  const section = await tauri.updateSection(params);
  $sections.set(replaceItem($sections.get(), section));
}

export async function deleteSection(id: string): Promise<void> {
  await tauri.deleteSection(id);
  $sections.set(removeItem($sections.get(), id));
}

export async function toggleSectionCollapsed(id: string): Promise<void> {
  const section = await tauri.toggleSectionCollapsed(id);
  $sections.set(replaceItem($sections.get(), section));
}

export async function toggleSectionArchived(id: string): Promise<void> {
  const section = await tauri.toggleSectionArchived(id);
  $sections.set(replaceItem($sections.get(), section));
}

export async function reorderSections(projectId: string, moves: ReorderMove[]): Promise<void> {
  await tauri.reorderSections(projectId, moves);
  const indexed = new Map(moves.map((m) => [m.id, m.index]));
  $sections.set(
    $sections.get().map((sec) => {
      const idx = indexed.get(sec.id);
      return idx !== undefined ? { ...sec, index: idx } : sec;
    }),
  );
}

// --- Tag actions ---

export async function createTag(params: CreateTagParams): Promise<Tag | null> {
  const tag = await tauri.createTag(params);
  if (tag) $tags.set([...$tags.get(), tag]);
  return tag;
}

export async function updateTag(params: UpdateTagParams): Promise<void> {
  const tag = await tauri.updateTag(params);
  $tags.set(replaceItem($tags.get(), tag));
}

export async function deleteTag(id: string): Promise<void> {
  await tauri.deleteTag(id);
  $tags.set(removeItem($tags.get(), id));
  $taskTags.set($taskTags.get().filter((tt) => tt.tagId !== id));
  const next = new Set($activeTagFilters.get());
  next.delete(id);
  $activeTagFilters.set(next);
}

export async function addTagToTask(taskId: string, tagId: string): Promise<void> {
  await tauri.addTagToTask(taskId, tagId);
  $taskTags.set([...$taskTags.get(), { taskId, tagId }]);
}

export async function removeTagFromTask(taskId: string, tagId: string): Promise<void> {
  await tauri.removeTagFromTask(taskId, tagId);
  $taskTags.set(
    $taskTags.get().filter((tt) => !(tt.taskId === taskId && tt.tagId === tagId)),
  );
}

// --- Checklist actions ---

export async function createChecklistItem(params: CreateChecklistItemParams): Promise<ChecklistItem> {
  const item = await tauri.createChecklistItem(params);
  $checklistItems.set([...$checklistItems.get(), item]);
  return item;
}

export async function updateChecklistItem(params: UpdateChecklistItemParams): Promise<void> {
  const item = await tauri.updateChecklistItem(params);
  $checklistItems.set(replaceItem($checklistItems.get(), item));
}

export async function toggleChecklistItem(id: string): Promise<void> {
  const item = await tauri.toggleChecklistItem(id);
  $checklistItems.set(replaceItem($checklistItems.get(), item));
}

export async function deleteChecklistItem(id: string): Promise<void> {
  await tauri.deleteChecklistItem(id);
  $checklistItems.set(removeItem($checklistItems.get(), id));
}

export async function reorderChecklistItems(taskId: string, moves: ReorderMove[]): Promise<void> {
  await tauri.reorderChecklistItems(taskId, moves);
  const indexed = new Map(moves.map((m) => [m.id, m.index]));
  $checklistItems.set(
    $checklistItems.get().map((ci) => {
      const idx = indexed.get(ci.id);
      return idx !== undefined ? { ...ci, index: idx } : ci;
    }),
  );
}

// --- Sync mutations ---

export async function initialSync(mode: 'fresh' | 'merge' | 'push'): Promise<void> {
  await tauri.initialSync(mode);
}
