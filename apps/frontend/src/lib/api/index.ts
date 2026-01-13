// API Client & Types
export {
	api,
	unwrap,
	type ApiResponse,
	type PaginatedResponse,
} from './client';

// Auth
export {
	login,
	register,
	getMe,
	logout,
	storeToken,
	isAuthenticated,
	type User,
	type LoginRequest,
	type RegisterRequest,
	type AuthResponse,
} from './auth';

// Tasks
export {
	getTasks,
	getTask,
	createTask,
	updateTask,
	deleteTask,
	getLastTaskEndTime,
	getTaskGoals,
	type Task,
	type TaskItem,
	type Quantity,
	type TaskGoalLink,
	type Category,
	type CreateTaskRequest,
	type UpdateTaskRequest,
	type TaskFilterParams,
	type TaskGoalsResponse,
} from './tasks';

// Goals
export {
	getGoals,
	getGoal,
	createGoal,
	updateGoal,
	deleteGoal,
	getTodayGoals,
	getGoalLogs,
	getGoalLogsSummary,
	getGoalTasks,
	type Goal,
	type Recurrence,
	type Target,
	type GoalTaskLink,
	type CreateGoalRequest,
	type GoalLog,
	type GoalLogEvent,
	type GoalLogsResponse,
	type GoalLogsSummary,
	type GoalTasksResponse,
	type TriggeringTaskInfo,
} from './goals';

// Emotions
export {
	getEmotionGrid,
	type Emotion,
	type InferredEmotion,
	type EmotionGridResponse,
} from './emotions';

// Categories
export {
	getCategories,
	getCategory,
	createCategory,
	updateCategory,
	deleteCategory,
	type CreateCategoryRequest,
	type UpdateCategoryRequest,
	type CategoryPageResponse,
} from './categories';

// Templates
export {
	getTemplates,
	getTemplate,
	createTemplate,
	updateTemplate,
	deleteTemplate,
	getQuickLogTemplates,
	type TaskTemplate,
	type CreateTemplateRequest,
	type UpdateTemplateRequest,
	type InstantiateTemplateRequest,
} from './templates';

// Task-Goal Links
export {
	getGoalsForTask,
	linkTaskToGoal,
	batchLinkTaskToGoals,
	updateTaskGoalLink,
	unlinkTaskFromGoal,
	getTasksForGoal,
	type TaskGoalWithGoal,
	type TaskGoalWithTask,
	type LinkTaskToGoalRequest,
	type BatchLinkRequest,
	type UpdateLinkRequest as UpdateTaskGoalLinkRequest,
	type GoalsForTaskResponse,
	type TasksForGoalResponse,
} from './taskGoals';

// Goal Actions
export {
	getGoalActions,
	createGoalAction,
	updateGoalAction,
	deleteGoalAction,
	reorderGoalActions,
	toggleActionComplete,
	type GoalAction,
	type CreateGoalActionRequest,
	type UpdateGoalActionRequest,
	type ReorderActionsRequest,
	type ActionListResponse,
} from './goalActions';

// Units
export {
	getUnits,
	getUnit,
	createUnit,
	updateUnit,
	deleteUnit,
	type Unit,
	type UnitListResponse,
	type CreateUnitRequest,
	type UpdateUnitRequest,
} from './units';
