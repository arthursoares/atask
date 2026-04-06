// Atoms
export { $tasks, $taskTags, $tagsByTaskId } from './tasks';
export { $projects, $projectTags, $tagsByProjectId, $activeProjects, useActiveProjects } from './projects';
export { $areas, $activeAreas, useActiveAreas } from './areas';
export { $sections, useSectionsForProject } from './sections';
export { $tags, useTagsForTask } from './tags';
export { $checklistItems, useChecklistForTask } from './checklist';
export { $activities, useActivitiesForTask } from './activities';
export { $locations } from './locations';
export { $taskLinks, $linksByTaskId } from './taskLinks';

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
  $taskPointerDrag,
  setActiveView,
  selectTask,
  clearSelectedTask,
  clearSelectedTasks,
  openTaskEditor,
  closeTaskEditor,
  toggleTaskSelection,
  selectTaskRange,
  toggleTagFilter,
  clearTagFilters,
  startTaskPointerDrag,
  endTaskPointerDrag,
  setTaskPointerHoverTarget,
  $syncStatus,
} from './ui';
export type { SyncStatusState, TaskPointerDragState } from './ui';

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
  cancelProject,
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
  addTagToProject,
  removeTagFromProject,
  createChecklistItem,
  updateChecklistItem,
  toggleChecklistItem,
  deleteChecklistItem,
  reorderChecklistItems,
  createActivity,
  createMutationActivity,
  addTaskLink,
  removeTaskLink,
  createLocation,
  updateLocation,
  deleteLocation,
  setTaskLocation,
  initialSync,
} from './mutations';
