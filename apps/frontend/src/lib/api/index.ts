// API Client & Types
export { api, unwrap, type ApiResponse, type PaginatedResponse } from './client';

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
    type AuthResponse
} from './auth';

// Tasks
export {
    getTasks,
    getTask,
    createTask,
    updateTask,
    deleteTask,
    type Task,
    type TaskItem,
    type Quantity,
    type TaskGoalLink,
    type Category,
    type InferredEmotion,
    type CreateTaskRequest,
    type UpdateTaskRequest
} from './tasks';

// Goals
export {
    getGoals,
    getGoal,
    createGoal,
    updateGoal,
    deleteGoal,
    getTodayGoals,
    type Goal,
    type Recurrence,
    type Target,
    type GoalTaskLink,
    type CreateGoalRequest
} from './goals';

// Emotions
export {
    getEmotionGrid,
    getEmotion,
    type Emotion,
    type GridEmotion,
    type EmotionGridResponse
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
    type CategoryPageResponse
} from './categories';
