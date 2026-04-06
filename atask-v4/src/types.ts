// Domain types — must match Rust models exactly (camelCase via serde rename_all)

export interface Task {
  id: string;
  title: string;
  notes: string;
  status: number;
  schedule: number;
  startDate: string | null;
  deadline: string | null;
  completedAt: string | null;
  index: number;
  todayIndex: number | null;
  timeSlot: string | null;
  projectId: string | null;
  sectionId: string | null;
  areaId: string | null;
  locationId: string | null;
  createdAt: string;
  updatedAt: string;
  syncStatus: number;
  repeatRule: string | null;
}

export interface Project {
  id: string;
  title: string;
  notes: string;
  status: number;
  color: string;
  areaId: string | null;
  index: number;
  completedAt: string | null;
  createdAt: string;
  updatedAt: string;
}

export interface Area {
  id: string;
  title: string;
  index: number;
  archived: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface Section {
  id: string;
  title: string;
  projectId: string;
  index: number;
  archived: boolean;
  collapsed: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface Tag {
  id: string;
  title: string;
  index: number;
  createdAt: string;
  updatedAt: string;
}

export interface TaskTag {
  taskId: string;
  tagId: string;
}

export interface TaskLink {
  taskId: string;
  linkedTaskId: string;
}

export interface ProjectTag {
  projectId: string;
  tagId: string;
}

export interface ChecklistItem {
  id: string;
  title: string;
  status: number;
  taskId: string;
  index: number;
  createdAt: string;
  updatedAt: string;
}

export interface Activity {
  id: string;
  taskId: string;
  actorId: string | null;
  actorType: 'human' | 'agent';
  type: string;
  content: string;
  createdAt: string;
}

export interface Location {
  id: string;
  name: string;
  latitude: number | null;
  longitude: number | null;
  radius: number | null;
  address: string | null;
  createdAt: string;
  updatedAt: string;
}

export interface AppState {
  tasks: Task[];
  projects: Project[];
  areas: Area[];
  sections: Section[];
  tags: Tag[];
  taskTags: TaskTag[];
  taskLinks: TaskLink[];
  projectTags: ProjectTag[];
  checklistItems: ChecklistItem[];
  activities: Activity[];
  locations: Location[];
}

// Repeat rule (stored as JSON string in repeatRule field)
export interface RepeatRule {
  type: "fixed" | "afterCompletion";
  interval: number;
  unit: "day" | "week" | "month" | "year";
}

// --- Param types for invoke commands ---

export interface CreateTaskParams {
  title: string;
  notes?: string;
  schedule?: number;
  startDate?: string;
  deadline?: string;
  timeSlot?: string;
  projectId?: string;
  sectionId?: string;
  areaId?: string;
  tagIds?: string[];
  repeatRule?: string;
}

export interface UpdateTaskParams {
  id: string;
  title?: string;
  notes?: string;
  schedule?: number;
  startDate?: string | null;
  deadline?: string | null;
  timeSlot?: string | null;
  projectId?: string | null;
  sectionId?: string | null;
  areaId?: string | null;
  repeatRule?: string | null;
  tagIds?: string[];
}

export interface CreateProjectParams {
  title: string;
  color?: string;
  areaId?: string;
}

export interface UpdateProjectParams {
  id: string;
  title?: string;
  notes?: string;
  color?: string;
  areaId?: string | null;
}

export interface CreateAreaParams {
  title: string;
}

export interface UpdateAreaParams {
  id: string;
  title: string;
}

export interface CreateSectionParams {
  title: string;
  projectId: string;
}

export interface UpdateSectionParams {
  id: string;
  title?: string;
}

export interface CreateTagParams {
  title: string;
}

export interface UpdateTagParams {
  id: string;
  title: string;
}

export interface CreateLocationParams {
  name: string;
}

export interface UpdateLocationParams {
  id: string;
  name: string;
}

export interface CreateChecklistItemParams {
  title: string;
  taskId: string;
}

export interface UpdateChecklistItemParams {
  id: string;
  title: string;
}

export interface ReorderMove {
  id: string;
  index: number;
}

// View types for store UI state
export type ActiveView =
  | "inbox"
  | "today"
  | "upcoming"
  | "someday"
  | "logbook"
  | "settings"
  | `project-${string}`
  | `area-${string}`;

// Settings types
export interface Settings {
  serverUrl: string;
  apiKey: string;
  syncEnabled: boolean;
}

export interface UpdateSettingsParams {
  serverUrl?: string;
  apiKey?: string;
  syncEnabled?: boolean;
}
