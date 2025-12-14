import { api, unwrap } from './client';

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

export interface GridEmotion {
    id: string;
    name: string;
    emoji: string;
    description: string;
    x: number;
    y: number;
    quadrant: string;
}

export interface EmotionGridResponse {
    yellow: GridEmotion[];
    green: GridEmotion[];
    red: GridEmotion[];
    blue: GridEmotion[];
    total: number;
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
