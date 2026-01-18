// Goal view mode - now only 'lane' or 'off'
export type GoalViewMode = 'lane' | 'off';

// Goal representation for timeline
export interface TimelineGoal {
	id: string;
	title: string;
	icon: string;
	color: string;
	startDate?: Date;
	deadline?: Date;
	recurrence?: {
		frequency: number;
		period: 'day' | 'week' | 'month';
	};
	progress: number;
	currentStreak: number;
	todayStatus: 'pending' | 'met' | 'exceeded' | undefined;
	linkedTaskIds: string[];
	target?: {
		value: number;
		currentValue: number;
		unitId: string;
	};
}

// Linked goal info on a task
export interface LinkedGoalInfo {
	id: string;
	title: string;
	icon?: string;
	color?: string;
	impactType: 'positive' | 'negative' | 'neutral';
	quantityValue?: number;
	unitSymbol?: string;
}

export interface TimelineTask {
	id: string;
	title: string;
	description?: string;
	startTime: Date;
	endTime: Date;
	categoryColor?: string;
	categoryName?: string;
	completed?: boolean;
	emoji?: string;
	categoryId?: string;
	// Emotion fields (user-selected)
	emotionId?: string;
	emotionName?: string;
	emotionEmoji?: string;
	emotionQuadrant?: 'yellow' | 'green' | 'red' | 'blue';
	emotionDescription?: string;
	// Inferred emotion (AI suggested from positives/negatives)
	inferredEmotionId?: string;
	inferredEmotionName?: string;
	inferredEmotionEmoji?: string;
	inferredEmotionQuadrant?: 'yellow' | 'green' | 'red' | 'blue';
	inferredEmotionDescription?: string;
	// Linked goals (optional)
	linkedGoals?: LinkedGoalInfo[];
}

export interface TimelineProps {
	tasks?: TimelineTask[];
	goals?: TimelineGoal[];
	selectedDate?: Date;
	onTaskClick?: (taskId: string) => void;
	onToggleComplete?: (taskId: string, completed: boolean) => void;
	onCategoryClick?: (categoryId: string) => void;
	onDateChange?: (date: Date) => void;
	onTaskTimeUpdate?: (taskId: string, startTime: Date, endTime: Date) => void;
	editMode?: boolean;
	onEditModeChange?: (enabled: boolean) => void;
	// Goal-related props
	showGoals?: boolean;
	onShowGoalsChange?: (show: boolean) => void;
	onGoalClick?: (goalId: string) => void;
	onCreateTaskFromGoal?: (goalId: string) => void;
}
