import * as tauri from '../hooks/useTauri';

// Sync hook — set by useSync to avoid circular imports.
// Called after every mutation so sync can flush + pull.
let onMutation: (() => void) | null = null;
export function setOnMutation(cb: () => void) { onMutation = cb; }

function notifySync() { onMutation?.(); }
import { $tasks, $taskTags } from './tasks';
import { $projects, $projectTags } from './projects';
import { $areas } from './areas';
import { $sections } from './sections';
import { $tags } from './tags';
import { $checklistItems } from './checklist';
import { $activities } from './activities';
import { $locations } from './locations';
import { $taskLinks } from './taskLinks';
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
  Location,
  CreateLocationParams,
  UpdateLocationParams,
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

function appendItem<T>(arr: T[], item: T): T[] {
  return [...arr, item];
}

function applyReorder<T extends { id: string; index: number }>(
  items: T[],
  moves: ReorderMove[],
): T[] {
  const indexed = new Map(moves.map((move) => [move.id, move.index]));
  return items.map((item) => {
    const index = indexed.get(item.id);
    return index !== undefined ? { ...item, index } : item;
  });
}

function clearTaskUiState(taskIds: Set<string>): void {
  if (taskIds.size === 0) return;
  if (taskIds.has($selectedTaskId.get() ?? '')) $selectedTaskId.set(null);
  if (taskIds.has($expandedTaskId.get() ?? '')) $expandedTaskId.set(null);
}

function removeTaskArtifacts(taskIds: Set<string>): void {
  if (taskIds.size === 0) return;
  $checklistItems.set($checklistItems.get().filter((item) => !taskIds.has(item.taskId)));
  $taskTags.set($taskTags.get().filter((taskTag) => !taskIds.has(taskTag.taskId)));
  $taskLinks.set($taskLinks.get().filter((link) => !taskIds.has(link.taskId) && !taskIds.has(link.linkedTaskId)));
  clearTaskUiState(taskIds);
}

function deleteTaskFromStores(taskId: string): void {
  $tasks.set(removeItem($tasks.get(), taskId));
  removeTaskArtifacts(new Set([taskId]));
}

function deleteProjectFromStores(projectId: string): void {
  const taskIds = new Set(
    $tasks.get().filter((task) => task.projectId === projectId).map((task) => task.id),
  );

  $projects.set(removeItem($projects.get(), projectId));
  $tasks.set($tasks.get().filter((task) => task.projectId !== projectId));
  $sections.set($sections.get().filter((section) => section.projectId !== projectId));
  $projectTags.set($projectTags.get().filter((pt) => pt.projectId !== projectId));
  removeTaskArtifacts(taskIds);

  if ($activeView.get() === `project-${projectId}`) {
    $activeView.set('inbox');
  }
}

function removeTagFromStores(tagId: string): void {
  $tags.set(removeItem($tags.get(), tagId));
  $taskTags.set($taskTags.get().filter((taskTag) => taskTag.tagId !== tagId));
  $projectTags.set($projectTags.get().filter((pt) => pt.tagId !== tagId));

  const next = new Set($activeTagFilters.get());
  next.delete(tagId);
  $activeTagFilters.set(next);
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
  $taskLinks.set(data.taskLinks);
  $projectTags.set(data.projectTags);
  $checklistItems.set(data.checklistItems);
  $activities.set(data.activities);
  $locations.set(data.locations);
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
  $tasks.set(appendItem($tasks.get(), task));
  notifySync();
  return task;
}

export async function completeTask(id: string): Promise<void> {
  const task = await tauri.completeTask(id);
  $tasks.set(replaceItem($tasks.get(), task));
  notifySync();
  createMutationActivity(id, 'Completed');
}

export async function cancelTask(id: string): Promise<void> {
  const task = await tauri.cancelTask(id);
  $tasks.set(replaceItem($tasks.get(), task));
  notifySync();
  createMutationActivity(id, 'Cancelled');
}

export async function reopenTask(id: string): Promise<void> {
  const task = await tauri.reopenTask(id);
  $tasks.set(replaceItem($tasks.get(), task));
  notifySync();
  createMutationActivity(id, 'Reopened');
}

export async function updateTask(params: UpdateTaskParams): Promise<void> {
  const oldTask = $tasks.get().find((t) => t.id === params.id);
  const task = await tauri.updateTask(params);
  $tasks.set(replaceItem($tasks.get(), task));
  notifySync();

  if (oldTask) {
    if (params.schedule !== undefined && params.schedule !== oldTask.schedule) {
      const labels: Record<number, string> = { 0: 'Inbox', 1: 'Today', 2: 'Someday', 3: 'Upcoming' };
      createMutationActivity(params.id, `Moved to ${labels[params.schedule] ?? 'unknown'}`);
    }
    if (params.projectId !== undefined && params.projectId !== oldTask.projectId) {
      if (params.projectId) {
        const project = $projects.get().find((p) => p.id === params.projectId);
        createMutationActivity(params.id, `Moved to ${project?.title ?? 'project'}`);
      } else {
        createMutationActivity(params.id, 'Removed from project');
      }
    }
  }
}

export async function duplicateTask(id: string): Promise<void> {
  const task = await tauri.duplicateTask(id);
  $tasks.set(appendItem($tasks.get(), task));
  notifySync();
}

export async function deleteTask(id: string): Promise<void> {
  await tauri.deleteTask(id);
  deleteTaskFromStores(id);
  notifySync();
}

export async function reorderTasks(moves: ReorderMove[]): Promise<void> {
  await tauri.reorderTasks(moves);
  $tasks.set(applyReorder($tasks.get(), moves));
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
  $projects.set(appendItem($projects.get(), project));
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

export async function cancelProject(id: string): Promise<void> {
  const project = await tauri.cancelProject(id);
  $projects.set(replaceItem($projects.get(), project));
  notifySync();
}

export async function reopenProject(id: string): Promise<void> {
  const project = await tauri.reopenProject(id);
  $projects.set(replaceItem($projects.get(), project));
}

export async function deleteProject(id: string): Promise<void> {
  await tauri.deleteProject(id);
  deleteProjectFromStores(id);
  notifySync();
}

export async function moveProjectToArea(id: string, areaId: string | null): Promise<void> {
  const project = await tauri.moveProjectToArea(id, areaId);
  $projects.set(replaceItem($projects.get(), project));
}

export async function reorderProjects(moves: ReorderMove[]): Promise<void> {
  await tauri.reorderProjects(moves);
  $projects.set(applyReorder($projects.get(), moves));
}

// --- Area actions ---

export async function createArea(params: CreateAreaParams): Promise<Area> {
  const area = await tauri.createArea(params);
  $areas.set(appendItem($areas.get(), area));
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
  $areas.set(applyReorder($areas.get(), moves));
}

// --- Section actions ---

export async function createSection(params: CreateSectionParams): Promise<Section> {
  const section = await tauri.createSection(params);
  $sections.set(appendItem($sections.get(), section));
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
  $sections.set(applyReorder($sections.get(), moves));
}

// --- Tag actions ---

export async function createTag(params: CreateTagParams): Promise<Tag | null> {
  const tag = await tauri.createTag(params);
  if (tag) $tags.set(appendItem($tags.get(), tag));
  return tag;
}

export async function updateTag(params: UpdateTagParams): Promise<void> {
  const tag = await tauri.updateTag(params);
  $tags.set(replaceItem($tags.get(), tag));
}

export async function deleteTag(id: string): Promise<void> {
  await tauri.deleteTag(id);
  removeTagFromStores(id);
}

export async function addTagToTask(taskId: string, tagId: string): Promise<void> {
  await tauri.addTagToTask(taskId, tagId);
  $taskTags.set(appendItem($taskTags.get(), { taskId, tagId }));
}

export async function removeTagFromTask(taskId: string, tagId: string): Promise<void> {
  await tauri.removeTagFromTask(taskId, tagId);
  $taskTags.set(
    $taskTags.get().filter((tt) => !(tt.taskId === taskId && tt.tagId === tagId)),
  );
}

export async function addTagToProject(projectId: string, tagId: string): Promise<void> {
  await tauri.addTagToProject(projectId, tagId);
  $projectTags.set(appendItem($projectTags.get(), { projectId, tagId }));
  notifySync();
}

export async function removeTagFromProject(projectId: string, tagId: string): Promise<void> {
  await tauri.removeTagFromProject(projectId, tagId);
  $projectTags.set(
    $projectTags.get().filter((pt) => !(pt.projectId === projectId && pt.tagId === tagId)),
  );
  notifySync();
}

// --- Task link actions ---

export async function addTaskLink(taskId: string, linkedTaskId: string): Promise<void> {
  await tauri.addTaskLink(taskId, linkedTaskId);
  $taskLinks.set(appendItem($taskLinks.get(), { taskId, linkedTaskId }));
  notifySync();
}

export async function removeTaskLink(taskId: string, linkedTaskId: string): Promise<void> {
  await tauri.removeTaskLink(taskId, linkedTaskId);
  $taskLinks.set(
    $taskLinks.get().filter((tl) => !(tl.taskId === taskId && tl.linkedTaskId === linkedTaskId)),
  );
  notifySync();
}

// --- Checklist actions ---

export async function createChecklistItem(params: CreateChecklistItemParams): Promise<ChecklistItem> {
  const item = await tauri.createChecklistItem(params);
  $checklistItems.set(appendItem($checklistItems.get(), item));
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
  $checklistItems.set(applyReorder($checklistItems.get(), moves));
}

// --- Activity actions ---

export async function createActivity(taskId: string, content: string): Promise<void> {
  const activity = await tauri.createActivityCommand(taskId, 'human', 'comment', content);
  $activities.set([...$activities.get(), activity]);
  notifySync();
}

export async function createMutationActivity(taskId: string, content: string): Promise<void> {
  const activity = await tauri.createActivityCommand(taskId, 'human', 'status_change', content);
  $activities.set([...$activities.get(), activity]);
}

// --- Sync mutations ---

// --- Location actions ---

export async function createLocation(params: CreateLocationParams): Promise<Location> {
  const location = await tauri.createLocation(params);
  $locations.set(appendItem($locations.get(), location));
  notifySync();
  return location;
}

export async function updateLocation(params: UpdateLocationParams): Promise<void> {
  const location = await tauri.updateLocation(params);
  $locations.set(replaceItem($locations.get(), location));
  notifySync();
}

export async function deleteLocation(id: string): Promise<void> {
  await tauri.deleteLocation(id);
  $locations.set(removeItem($locations.get(), id));
  // Clear locationId on tasks that referenced this location
  $tasks.set($tasks.get().map((t) => (t.locationId === id ? { ...t, locationId: null } : t)));
  notifySync();
}

export async function setTaskLocation(taskId: string, locationId: string | null): Promise<void> {
  const task = await tauri.setTaskLocation(taskId, locationId);
  $tasks.set(replaceItem($tasks.get(), task));
  notifySync();
}

// --- Sync mutations ---

export async function initialSync(mode: 'fresh' | 'merge' | 'push'): Promise<void> {
  await tauri.initialSync(mode);
}
