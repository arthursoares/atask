// Atoms
export { $tasks, $taskTags, $tagsByTaskId } from './tasks';
export { $projects, $activeProjects, useActiveProjects } from './projects';
export { $areas, $activeAreas, useActiveAreas } from './areas';
export { $sections, useSectionsForProject } from './sections';
export { $tags, useTagsForTask } from './tags';
export { $checklistItems, useChecklistForTask } from './checklist';

// UI atoms
export {
  $activeView,
  $selectedTaskId,
  $selectedTaskIds,
  $expandedTaskId,
  $showPalette,
  $showQuickMove,
  $showSearch,
  $showSidebar,
  $showShortcuts,
  $activeTagFilters,
  toggleTagFilter,
  clearTagFilters,
  $syncStatus,
} from './ui';
export type { SyncStatusState } from './ui';

// Selectors
export {
  $inbox,
  $today,
  $todayMorning,
  $todayEvening,
  $upcoming,
  $someday,
  $logbook,
  useInbox,
  useTodayMorning,
  useTodayEvening,
  useUpcoming,
  useSomeday,
  useLogbook,
  useTasksForProject,
} from './selectors';
export type { UpcomingGroup } from './selectors';

// Mutations
export {
  loadAll,
  createTask,
  completeTask,
  cancelTask,
  reopenTask,
  updateTask,
  duplicateTask,
  deleteTask,
  reorderTasks,
  setTodayIndex,
  moveTaskToSection,
  createProject,
  updateProject,
  completeProject,
  reopenProject,
  deleteProject,
  moveProjectToArea,
  reorderProjects,
  createArea,
  updateArea,
  deleteArea,
  toggleAreaArchived,
  reorderAreas,
  createSection,
  updateSection,
  deleteSection,
  toggleSectionCollapsed,
  toggleSectionArchived,
  reorderSections,
  createTag,
  updateTag,
  deleteTag,
  addTagToTask,
  removeTagFromTask,
  createChecklistItem,
  updateChecklistItem,
  toggleChecklistItem,
  deleteChecklistItem,
  reorderChecklistItems,
  initialSync,
} from './mutations';
