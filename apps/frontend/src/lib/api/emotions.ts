import { api, unwrap } from './client';
import type { TaskItem } from './tasks';

// =============================================================================
// EMOTION TYPES (matching Go backend - 100 emotions in 4 quadrants)
// =============================================================================

export interface Emotion {
    id: string;
    name: string;
    emoji: string;
    quadrant: 'yellow' | 'green' | 'red' | 'blue';
    x: number;
    y: number;
    valence: number;
    arousal: number;
    dominance: number;
    intensity: number;
    certainty: number;
    social: number;
    description: string;
}

export interface InferredEmotion {
    valence: number;
    arousal: number;
    dominance: number;
    quadrant: string;
    closest_emotion_id: string;
    closest_emotion_name: string;
    positive_count: number;
    negative_count: number;
    dissonance: number;
}

export interface EmotionGridResponse {
    yellow: Emotion[];
    green: Emotion[];
    red: Emotion[];
    blue: Emotion[];
    total: number;
}

export interface InferEmotionRequest {
    positives: TaskItem[];
    negatives: TaskItem[];
}

export interface InferEmotionResponse {
    inferred_emotion: InferredEmotion | null;
}



// =============================================================================
// EMOTION API FUNCTIONS
// =============================================================================

export async function getEmotionGrid(): Promise<EmotionGridResponse> {
    return unwrap(api.get('emotions/grid'));
}

export async function getEmotion(id: string): Promise<Emotion> {
    return unwrap(api.get(`emotions/${id}`));
}

export async function inferEmotion(request: InferEmotionRequest): Promise<InferEmotionResponse> {
    return unwrap(api.post('emotions/infer', { json: request }));
}


