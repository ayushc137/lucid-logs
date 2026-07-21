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
	getGoalDailyProgress,
	getGoalPeriodStats,
	getGoalStreakHistory,
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
	type DailyStats,
	type PeriodSnapshot,
	type StreakEvent,
	type DailyProgressResponse,
	type StreakAnalytics,
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

// Activities
export {
	getActivities,
	getPinnedActivities,
	getActivity,
	createActivity,
	updateActivity,
	deleteActivity,
	instantLog,
	scheduleActivity,
	getActivityLinkedGoals,
	linkGoalToActivity,
	unlinkGoalFromActivity,
	type Activity,
	type ActivityGoalLink,
	type TimerSession,
	type TimerBreak,
	type CreateActivityRequest,
	type UpdateActivityRequest,
	type GoalLinkInput as ActivityGoalLinkInput,
	type InstantLogRequest,
	type InstantLogResponse,
	type GoalUpdateSummary,
	type ScheduleRequest,
	type ScheduleResponse,
	type TaskDefaults,
	type GoalLinkDefault,
	type ActivityGoalLinkDetail,
} from './activities';

// Retrospectives
export {
	getRetrospectives,
	getRetrospective,
	generateRetrospective,
	updateRetrospective,
	deleteRetrospective,
	type Retrospective,
	type RetroAutoSummary,
	type MoodSummary,
	type QuadrantDist,
	type MoodEvent,
	type HabitsSummary,
	type HabitStatus,
	type StreaksSummary,
	type StreakUpdate,
	type TasksSummary,
	type CategoryCount,
	type GoalsSummary,
	type GoalImpact,
	type GoalHighlight,
	type CategoriesSummary,
	type CategoryTime,
	type NeglectedArea,
	type UserReflection,
	type GenerateRetroRequest,
	type UpdateRetroRequest,
	type RetroListResponse,
} from './retrospectives';

// Analytics
export {
	getAnalyticsDashboard,
	getAnalyticsStreaks,
	getActivityHeatmap,
	type DashboardResponse,
	type TaskMetrics,
	type EmotionMetrics,
	type GoalMetrics,
	type CategoryMetrics,
	type CategoryBreakdown,
	type EmotionCount,
	type DailyMood,
	type GoalProgressItem,
	type StreakInfo,
	type StreaksResponse,
	type ActivityHeatmapDay,
	type ActivityHeatmapResponse,
} from './analytics';
