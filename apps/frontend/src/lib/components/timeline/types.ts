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
}

export interface TimelineProps {
    tasks?: TimelineTask[];
    selectedDate?: Date;
    onTaskClick?: (taskId: string) => void;
    onToggleComplete?: (taskId: string, completed: boolean) => void;
    onCategoryClick?: (categoryId: string) => void;
    onDateChange?: (date: Date) => void;
    onTaskTimeUpdate?: (taskId: string, startTime: Date, endTime: Date) => void;
    editMode?: boolean;
    onEditModeChange?: (enabled: boolean) => void;
}
