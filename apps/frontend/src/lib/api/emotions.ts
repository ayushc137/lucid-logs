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



export interface EmotionGridResponse {
    yellow: Emotion[];
    green: Emotion[];
    red: Emotion[];
    blue: Emotion[];
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


