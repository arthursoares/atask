// Typed invoke() wrappers for all Tauri commands

import { invoke } from "@tauri-apps/api/core";
import type {
  AppState,
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
  Settings,
  UpdateSettingsParams,
} from "../types";

// --- Data loading ---

export function loadAll(): Promise<AppState> {
  return invoke<AppState>("load_all");
}

// --- Task commands ---

export function createTask(params: CreateTaskParams): Promise<Task> {
  return invoke<Task>("create_task", { params });
}

export function completeTask(id: string): Promise<Task> {
  return invoke<Task>("complete_task", { id });
}

export function cancelTask(id: string): Promise<Task> {
  return invoke<Task>("cancel_task", { id });
}

export function reopenTask(id: string): Promise<Task> {
  return invoke<Task>("reopen_task", { id });
}

export function updateTask(params: UpdateTaskParams): Promise<Task> {
  return invoke<Task>("update_task", { params });
}

export function duplicateTask(id: string): Promise<Task> {
  return invoke<Task>("duplicate_task", { id });
}

export function deleteTask(id: string): Promise<void> {
  return invoke<void>("delete_task", { id });
}

export function reorderTasks(moves: ReorderMove[]): Promise<void> {
  return invoke<void>("reorder_tasks", { moves });
}

export function setTodayIndex(id: string, index: number): Promise<Task> {
  return invoke<Task>("set_today_index", { id, index });
}

export function moveTaskToSection(
  taskId: string,
  sectionId: string | null,
): Promise<Task> {
  return invoke<Task>("move_task_to_section", { taskId, sectionId });
}

// --- Project commands ---

export function createProject(params: CreateProjectParams): Promise<Project> {
  return invoke<Project>("create_project", { params });
}

export function updateProject(params: UpdateProjectParams): Promise<Project> {
  return invoke<Project>("update_project", { params });
}

export function completeProject(id: string): Promise<Project> {
  return invoke<Project>("complete_project", { id });
}

export function reopenProject(id: string): Promise<Project> {
  return invoke<Project>("reopen_project", { id });
}

export function deleteProject(id: string): Promise<void> {
  return invoke<void>("delete_project", { id });
}

export function moveProjectToArea(
  projectId: string,
  areaId: string | null,
): Promise<Project> {
  return invoke<Project>("move_project_to_area", { projectId, areaId });
}

export function reorderProjects(moves: ReorderMove[]): Promise<void> {
  return invoke<void>("reorder_projects", { moves });
}

// --- Area commands ---

export function createArea(params: CreateAreaParams): Promise<Area> {
  return invoke<Area>("create_area", { params });
}

export function updateArea(params: UpdateAreaParams): Promise<Area> {
  return invoke<Area>("update_area", { params });
}

export function deleteArea(id: string): Promise<void> {
  return invoke<void>("delete_area", { id });
}

export function toggleAreaArchived(id: string): Promise<Area> {
  return invoke<Area>("toggle_area_archived", { id });
}

export function reorderAreas(moves: ReorderMove[]): Promise<void> {
  return invoke<void>("reorder_areas", { moves });
}

// --- Section commands ---

export function createSection(params: CreateSectionParams): Promise<Section> {
  return invoke<Section>("create_section", { params });
}

export function updateSection(params: UpdateSectionParams): Promise<Section> {
  return invoke<Section>("update_section", { params });
}

export function deleteSection(id: string): Promise<void> {
  return invoke<void>("delete_section", { id });
}

export function toggleSectionCollapsed(id: string): Promise<Section> {
  return invoke<Section>("toggle_section_collapsed", { id });
}

export function toggleSectionArchived(id: string): Promise<Section> {
  return invoke<Section>("toggle_section_archived", { id });
}

export function reorderSections(
  projectId: string,
  moves: ReorderMove[],
): Promise<void> {
  return invoke<void>("reorder_sections", { projectId, moves });
}

// --- Tag commands ---

export function createTag(params: CreateTagParams): Promise<Tag | null> {
  return invoke<Tag | null>("create_tag", { params });
}

export function updateTag(params: UpdateTagParams): Promise<Tag> {
  return invoke<Tag>("update_tag", { params });
}

export function deleteTag(id: string): Promise<void> {
  return invoke<void>("delete_tag", { id });
}

export function addTagToTask(taskId: string, tagId: string): Promise<void> {
  return invoke<void>("add_tag_to_task", { taskId, tagId });
}

export function removeTagFromTask(
  taskId: string,
  tagId: string,
): Promise<void> {
  return invoke<void>("remove_tag_from_task", { taskId, tagId });
}

// --- Checklist commands ---

export function createChecklistItem(
  params: CreateChecklistItemParams,
): Promise<ChecklistItem> {
  return invoke<ChecklistItem>("create_checklist_item", { params });
}

export function updateChecklistItem(
  params: UpdateChecklistItemParams,
): Promise<ChecklistItem> {
  return invoke<ChecklistItem>("update_checklist_item", { params });
}

export function toggleChecklistItem(id: string): Promise<ChecklistItem> {
  return invoke<ChecklistItem>("toggle_checklist_item", { id });
}

export function deleteChecklistItem(id: string): Promise<void> {
  return invoke<void>("delete_checklist_item", { id });
}

export function reorderChecklistItems(
  taskId: string,
  moves: ReorderMove[],
): Promise<void> {
  return invoke<void>("reorder_checklist_items", { taskId, moves });
}

// --- Settings commands ---

export function getSettings(): Promise<Settings> {
  return invoke<Settings>("get_settings");
}

export function updateSettings(params: UpdateSettingsParams): Promise<Settings> {
  return invoke<Settings>("update_settings", { params });
}

// --- Sync commands ---

export interface SyncStatus {
  isSyncing: boolean;
  lastSyncAt: string | null;
  lastError: string | null;
  pendingOpsCount: number;
}

export function getSyncStatus(): Promise<SyncStatus> {
  return invoke<SyncStatus>("get_sync_status");
}

export function triggerSync(): Promise<void> {
  return invoke<void>("trigger_sync");
}

export function testConnection(): Promise<boolean> {
  return invoke<boolean>("test_connection");
}

export function initialSync(mode: "fresh" | "merge" | "push"): Promise<void> {
  return invoke<void>("initial_sync", { params: { mode } });
}
