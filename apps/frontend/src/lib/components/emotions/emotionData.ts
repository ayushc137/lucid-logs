// =============================================================================
// EMOTION DATA & UTILITIES
// Hardcoded emotion data from emotions.json for frontend use
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

export type Quadrant = 'yellow' | 'green' | 'red' | 'blue';

// Quadrant color configurations with gradient stops
export const QUADRANT_COLORS: Record<Quadrant, {
    primary: string;
    secondary: string;
    accent: string;
    glow: string;
    gradient: string;
}> = {
    yellow: {
        primary: '#FBBF24',    // Amber-400
        secondary: '#F59E0B',  // Amber-500
        accent: '#FDE68A',     // Amber-200
        glow: 'rgba(251, 191, 36, 0.4)',
        gradient: 'linear-gradient(135deg, #FDE68A 0%, #FBBF24 50%, #F59E0B 100%)'
    },
    green: {
        primary: '#34D399',    // Emerald-400
        secondary: '#10B981',  // Emerald-500
        accent: '#A7F3D0',     // Emerald-200
        glow: 'rgba(52, 211, 153, 0.4)',
        gradient: 'linear-gradient(135deg, #A7F3D0 0%, #34D399 50%, #10B981 100%)'
    },
    red: {
        primary: '#F87171',    // Red-400
        secondary: '#EF4444',  // Red-500
        accent: '#FECACA',     // Red-200
        glow: 'rgba(248, 113, 113, 0.4)',
        gradient: 'linear-gradient(135deg, #FECACA 0%, #F87171 50%, #EF4444 100%)'
    },
    blue: {
        primary: '#60A5FA',    // Blue-400
        secondary: '#3B82F6',  // Blue-500
        accent: '#BFDBFE',     // Blue-200
        glow: 'rgba(96, 165, 250, 0.4)',
        gradient: 'linear-gradient(135deg, #BFDBFE 0%, #60A5FA 50%, #3B82F6 100%)'
    }
};

// Quadrant metadata for labels and descriptions
export const QUADRANT_META: Record<Quadrant, {
    label: string;
    energyLabel: string;
    pleasantnessLabel: string;
}> = {
    yellow: { label: 'High Energy Positive', energyLabel: 'High', pleasantnessLabel: 'Pleasant' },
    green: { label: 'Low Energy Positive', energyLabel: 'Low', pleasantnessLabel: 'Pleasant' },
    red: { label: 'High Energy Negative', energyLabel: 'High', pleasantnessLabel: 'Unpleasant' },
    blue: { label: 'Low Energy Negative', energyLabel: 'Low', pleasantnessLabel: 'Unpleasant' }
};

// Seeded random number generator for consistent procedural shapes
export function seededRandom(seed: string): () => number {
    let hash = 0;
    for (let i = 0; i < seed.length; i++) {
        const char = seed.charCodeAt(i);
        hash = ((hash << 5) - hash) + char;
        hash = hash & hash;
    }
    return function () {
        hash = Math.sin(hash) * 10000;
        return hash - Math.floor(hash);
    };
}

// Calculate normalized position within quadrant (0-1 range)
export function getNormalizedPosition(emotion: Emotion): { nx: number; ny: number } {
    const absX = Math.abs(emotion.x);
    const absY = Math.abs(emotion.y);
    return {
        nx: (absX - 1) / 4,  // 1-5 -> 0-1
        ny: (absY - 1) / 4   // 1-5 -> 0-1
    };
}

// Get intensity factor based on distance from center (0-1)
export function getIntensityFactor(emotion: Emotion): number {
    const { nx, ny } = getNormalizedPosition(emotion);
    return Math.sqrt(nx * nx + ny * ny) / Math.sqrt(2);
}

// Hardcoded emotions data
export const EMOTIONS: Emotion[] = [
    // Yellow Quadrant (High Energy Positive) - x: 1-5, y: 1-5
    { id: "E01", name: "Ecstatic", emoji: "🤩", quadrant: "yellow", x: 1, y: 5, valence: 0.95, arousal: 0.95, dominance: 0.75, intensity: 1.0, certainty: 0.6, social: 0.2, description: "Over-the-moon happy, can't contain your joy" },
    { id: "E02", name: "Elated", emoji: "🥳", quadrant: "yellow", x: 2, y: 5, valence: 0.9, arousal: 0.9, dominance: 0.7, intensity: 0.95, certainty: 0.55, social: 0.25, description: "Thrilled and celebrating inside" },
    { id: "E03", name: "Excited", emoji: "🎉", quadrant: "yellow", x: 3, y: 5, valence: 0.85, arousal: 0.88, dominance: 0.65, intensity: 0.9, certainty: 0.4, social: 0.15, description: "Pumped up about something coming" },
    { id: "E04", name: "Thrilled", emoji: "🎢", quadrant: "yellow", x: 4, y: 5, valence: 0.88, arousal: 0.92, dominance: 0.6, intensity: 0.92, certainty: 0.35, social: 0.2, description: "Heart racing with anticipation" },
    { id: "E05", name: "Energized", emoji: "⚡", quadrant: "yellow", x: 5, y: 5, valence: 0.75, arousal: 0.9, dominance: 0.72, intensity: 0.85, certainty: 0.55, social: 0.0, description: "Buzzing with energy" },
    { id: "E06", name: "Joyful", emoji: "😁", quadrant: "yellow", x: 1, y: 4, valence: 0.9, arousal: 0.75, dominance: 0.7, intensity: 0.85, certainty: 0.5, social: 0.3, description: "Warm, deep happiness from within" },
    { id: "E07", name: "Delighted", emoji: "🌈", quadrant: "yellow", x: 2, y: 4, valence: 0.88, arousal: 0.72, dominance: 0.68, intensity: 0.82, certainty: 0.55, social: 0.35, description: "Pleasantly surprised and pleased" },
    { id: "E08", name: "Passionate", emoji: "❤️‍🔥", quadrant: "yellow", x: 3, y: 4, valence: 0.85, arousal: 0.88, dominance: 0.75, intensity: 0.9, certainty: 0.6, social: 0.3, description: "Deeply invested and engaged" },
    { id: "E09", name: "Enthusiastic", emoji: "🙌", quadrant: "yellow", x: 4, y: 4, valence: 0.82, arousal: 0.82, dominance: 0.68, intensity: 0.8, certainty: 0.5, social: 0.35, description: "Eager and ready to dive in" },
    { id: "E10", name: "Amazed", emoji: "🤯", quadrant: "yellow", x: 5, y: 4, valence: 0.82, arousal: 0.8, dominance: 0.4, intensity: 0.85, certainty: -0.45, social: 0.4, description: "Blown away by something incredible" },
    { id: "E11", name: "Proud", emoji: "🏆", quadrant: "yellow", x: 1, y: 3, valence: 0.85, arousal: 0.7, dominance: 0.9, intensity: 0.8, certainty: 0.75, social: -0.2, description: "Accomplished something meaningful" },
    { id: "E12", name: "Confident", emoji: "😎", quadrant: "yellow", x: 2, y: 3, valence: 0.8, arousal: 0.65, dominance: 0.92, intensity: 0.75, certainty: 0.85, social: -0.15, description: "Sure of yourself and your abilities" },
    { id: "E13", name: "Triumphant", emoji: "🥇", quadrant: "yellow", x: 3, y: 3, valence: 0.88, arousal: 0.78, dominance: 0.95, intensity: 0.88, certainty: 0.9, social: -0.1, description: "Won! Achieved a victory" },
    { id: "E14", name: "Determined", emoji: "💪", quadrant: "yellow", x: 4, y: 3, valence: 0.7, arousal: 0.85, dominance: 0.9, intensity: 0.85, certainty: 0.8, social: -0.2, description: "Nothing will stop you" },
    { id: "E15", name: "Motivated", emoji: "🔥", quadrant: "yellow", x: 5, y: 3, valence: 0.75, arousal: 0.8, dominance: 0.78, intensity: 0.78, certainty: 0.65, social: -0.1, description: "Driven to take action now" },
    { id: "E16", name: "Happy", emoji: "😊", quadrant: "yellow", x: 1, y: 2, valence: 0.8, arousal: 0.6, dominance: 0.65, intensity: 0.7, certainty: 0.6, social: 0.25, description: "Simply feeling good right now" },
    { id: "E17", name: "Cheerful", emoji: "😃", quadrant: "yellow", x: 2, y: 2, valence: 0.78, arousal: 0.65, dominance: 0.62, intensity: 0.68, certainty: 0.55, social: 0.4, description: "Upbeat and positive mood" },
    { id: "E18", name: "Inspired", emoji: "💡", quadrant: "yellow", x: 3, y: 2, valence: 0.85, arousal: 0.75, dominance: 0.72, intensity: 0.82, certainty: 0.45, social: 0.4, description: "Full of new ideas and motivation" },
    { id: "E19", name: "Accomplished", emoji: "✅", quadrant: "yellow", x: 4, y: 2, valence: 0.82, arousal: 0.55, dominance: 0.88, intensity: 0.72, certainty: 0.8, social: -0.15, description: "Finished something you worked hard on" },
    { id: "E20", name: "Playful", emoji: "😜", quadrant: "yellow", x: 5, y: 2, valence: 0.78, arousal: 0.75, dominance: 0.55, intensity: 0.7, certainty: 0.4, social: 0.5, description: "In a fun, silly mood" },
    { id: "E21", name: "Optimistic", emoji: "🌟", quadrant: "yellow", x: 1, y: 1, valence: 0.78, arousal: 0.58, dominance: 0.65, intensity: 0.62, certainty: 0.55, social: 0.15, description: "Expecting things to work out well" },
    { id: "E22", name: "Hopeful", emoji: "🌅", quadrant: "yellow", x: 2, y: 1, valence: 0.75, arousal: 0.55, dominance: 0.5, intensity: 0.65, certainty: -0.35, social: 0.2, description: "Believing good things are coming" },
    { id: "E23", name: "Curious", emoji: "🧐", quadrant: "yellow", x: 3, y: 1, valence: 0.7, arousal: 0.65, dominance: 0.55, intensity: 0.65, certainty: -0.3, social: 0.25, description: "Want to explore and learn more" },
    { id: "E24", name: "Fascinated", emoji: "🔍", quadrant: "yellow", x: 4, y: 1, valence: 0.78, arousal: 0.72, dominance: 0.5, intensity: 0.75, certainty: -0.25, social: 0.35, description: "Captivated by something interesting" },
    { id: "E25", name: "Amused", emoji: "😆", quadrant: "yellow", x: 5, y: 1, valence: 0.75, arousal: 0.65, dominance: 0.6, intensity: 0.68, certainty: 0.5, social: 0.45, description: "Something made you smile or laugh" },

    // Green Quadrant (Low Energy Positive) - x: 1-5, y: -1 to -5
    { id: "E26", name: "Content", emoji: "☺️", quadrant: "green", x: 1, y: -1, valence: 0.82, arousal: -0.5, dominance: 0.72, intensity: 0.6, certainty: 0.7, social: 0.1, description: "Satisfied with how things are" },
    { id: "E27", name: "Satisfied", emoji: "👌", quadrant: "green", x: 2, y: -1, valence: 0.85, arousal: -0.45, dominance: 0.75, intensity: 0.65, certainty: 0.75, social: 0.1, description: "Got what you needed" },
    { id: "E28", name: "Fulfilled", emoji: "✨", quadrant: "green", x: 3, y: -1, valence: 0.9, arousal: -0.4, dominance: 0.82, intensity: 0.78, certainty: 0.8, social: 0.15, description: "Life feels complete and meaningful" },
    { id: "E29", name: "Relieved", emoji: "😮‍💨", quadrant: "green", x: 4, y: -1, valence: 0.75, arousal: -0.4, dominance: 0.62, intensity: 0.7, certainty: 0.65, social: 0.1, description: "A weight has been lifted" },
    { id: "E30", name: "Refreshed", emoji: "🌿", quadrant: "green", x: 5, y: -1, valence: 0.78, arousal: -0.35, dominance: 0.7, intensity: 0.65, certainty: 0.68, social: 0.05, description: "Renewed and ready again" },
    { id: "E31", name: "Grateful", emoji: "🙏", quadrant: "green", x: 1, y: -2, valence: 0.88, arousal: -0.35, dominance: 0.55, intensity: 0.75, certainty: 0.7, social: 0.8, description: "Thankful for what you have" },
    { id: "E32", name: "Thankful", emoji: "💝", quadrant: "green", x: 2, y: -2, valence: 0.85, arousal: -0.38, dominance: 0.52, intensity: 0.72, certainty: 0.68, social: 0.82, description: "Appreciating someone or something" },
    { id: "E33", name: "Blessed", emoji: "🌸", quadrant: "green", x: 3, y: -2, valence: 0.82, arousal: -0.4, dominance: 0.5, intensity: 0.68, certainty: 0.65, social: 0.78, description: "Feeling fortunate and lucky" },
    { id: "E34", name: "Connected", emoji: "🤝", quadrant: "green", x: 4, y: -2, valence: 0.85, arousal: -0.32, dominance: 0.55, intensity: 0.72, certainty: 0.55, social: 0.88, description: "Feeling close to others" },
    { id: "E35", name: "Belonging", emoji: "🏡", quadrant: "green", x: 5, y: -2, valence: 0.88, arousal: -0.35, dominance: 0.58, intensity: 0.75, certainty: 0.6, social: 0.9, description: "Part of something bigger than yourself" },
    { id: "E36", name: "Loving", emoji: "🥰", quadrant: "green", x: 1, y: -3, valence: 0.95, arousal: -0.3, dominance: 0.55, intensity: 0.85, certainty: 0.6, social: 0.95, description: "Heart full of love for someone" },
    { id: "E37", name: "Affectionate", emoji: "💕", quadrant: "green", x: 2, y: -3, valence: 0.88, arousal: -0.35, dominance: 0.52, intensity: 0.78, certainty: 0.58, social: 0.9, description: "Wanting to show warmth and care" },
    { id: "E38", name: "Tender", emoji: "💗", quadrant: "green", x: 3, y: -3, valence: 0.82, arousal: -0.42, dominance: 0.48, intensity: 0.72, certainty: 0.55, social: 0.85, description: "Soft, gentle feelings for someone" },
    { id: "E39", name: "Compassionate", emoji: "🤲", quadrant: "green", x: 4, y: -3, valence: 0.8, arousal: -0.3, dominance: 0.45, intensity: 0.7, certainty: 0.5, social: 0.92, description: "Caring deeply about others' wellbeing" },
    { id: "E40", name: "Thoughtful", emoji: "🤔", quadrant: "green", x: 5, y: -3, valence: 0.65, arousal: -0.4, dominance: 0.62, intensity: 0.55, certainty: 0.45, social: 0.35, description: "Reflecting and considering things" },
    { id: "E41", name: "Safe", emoji: "🔒", quadrant: "green", x: 1, y: -4, valence: 0.8, arousal: -0.55, dominance: 0.85, intensity: 0.62, certainty: 0.82, social: 0.15, description: "Protected and out of harm's way" },
    { id: "E42", name: "Secure", emoji: "🛡️", quadrant: "green", x: 2, y: -4, valence: 0.82, arousal: -0.52, dominance: 0.88, intensity: 0.65, certainty: 0.85, social: 0.12, description: "Stable and confident in your situation" },
    { id: "E43", name: "Comfortable", emoji: "🏠", quadrant: "green", x: 3, y: -4, valence: 0.78, arousal: -0.58, dominance: 0.72, intensity: 0.58, certainty: 0.72, social: 0.08, description: "At home and at ease" },
    { id: "E44", name: "Calm", emoji: "😌", quadrant: "green", x: 4, y: -4, valence: 0.8, arousal: -0.65, dominance: 0.72, intensity: 0.6, certainty: 0.72, social: 0.0, description: "Relaxed and in control" },
    { id: "E45", name: "Relaxed", emoji: "🛋️", quadrant: "green", x: 5, y: -4, valence: 0.78, arousal: -0.7, dominance: 0.68, intensity: 0.58, certainty: 0.68, social: 0.05, description: "Body and mind at ease" },
    { id: "E46", name: "Serene", emoji: "🧘", quadrant: "green", x: 1, y: -5, valence: 0.9, arousal: -0.8, dominance: 0.75, intensity: 0.7, certainty: 0.8, social: 0.0, description: "Deep inner peace and stillness" },
    { id: "E47", name: "Tranquil", emoji: "🌊", quadrant: "green", x: 2, y: -5, valence: 0.88, arousal: -0.82, dominance: 0.72, intensity: 0.65, certainty: 0.78, social: 0.0, description: "Like still water, completely calm" },
    { id: "E48", name: "Peaceful", emoji: "☮️", quadrant: "green", x: 3, y: -5, valence: 0.85, arousal: -0.75, dominance: 0.7, intensity: 0.65, certainty: 0.75, social: 0.05, description: "No worries, everything feels okay" },
    { id: "E49", name: "Rested", emoji: "🔋", quadrant: "green", x: 4, y: -5, valence: 0.75, arousal: -0.45, dominance: 0.72, intensity: 0.62, certainty: 0.7, social: 0.05, description: "Recovered and recharged" },
    { id: "E50", name: "Mellow", emoji: "🍃", quadrant: "green", x: 5, y: -5, valence: 0.72, arousal: -0.68, dominance: 0.62, intensity: 0.52, certainty: 0.62, social: 0.1, description: "Easygoing and laid-back" },

    // Red Quadrant (High Energy Negative) - x: -5 to -1, y: 1-5
    { id: "E51", name: "Enraged", emoji: "🤬", quadrant: "red", x: -5, y: 5, valence: -0.95, arousal: 0.98, dominance: 0.85, intensity: 1.0, certainty: 0.8, social: 0.45, description: "Furious beyond words" },
    { id: "E52", name: "Furious", emoji: "😡", quadrant: "red", x: -4, y: 5, valence: -0.9, arousal: 0.95, dominance: 0.8, intensity: 0.95, certainty: 0.75, social: 0.4, description: "Extremely angry, ready to explode" },
    { id: "E53", name: "Terrified", emoji: "😱", quadrant: "red", x: -3, y: 5, valence: -0.92, arousal: 0.98, dominance: -0.9, intensity: 1.0, certainty: -0.85, social: -0.3, description: "Extreme fear, frozen or panicking" },
    { id: "E54", name: "Panicked", emoji: "🆘", quadrant: "red", x: -2, y: 5, valence: -0.88, arousal: 0.95, dominance: -0.85, intensity: 0.95, certainty: -0.8, social: -0.25, description: "Everything feels out of control" },
    { id: "E55", name: "Overwhelmed", emoji: "😵", quadrant: "red", x: -1, y: 5, valence: -0.78, arousal: 0.88, dominance: -0.82, intensity: 0.9, certainty: -0.75, social: -0.2, description: "Too much to handle right now" },
    { id: "E56", name: "Angry", emoji: "😠", quadrant: "red", x: -5, y: 4, valence: -0.8, arousal: 0.88, dominance: 0.65, intensity: 0.85, certainty: 0.7, social: 0.35, description: "Mad at someone or something" },
    { id: "E57", name: "Scared", emoji: "😨", quadrant: "red", x: -4, y: 4, valence: -0.78, arousal: 0.82, dominance: -0.72, intensity: 0.82, certainty: -0.65, social: -0.15, description: "Something feels threatening" },
    { id: "E58", name: "Frightened", emoji: "😧", quadrant: "red", x: -3, y: 4, valence: -0.75, arousal: 0.78, dominance: -0.68, intensity: 0.78, certainty: -0.6, social: -0.12, description: "Startled or alarmed by danger" },
    { id: "E59", name: "Stressed", emoji: "🤯", quadrant: "red", x: -2, y: 4, valence: -0.7, arousal: 0.82, dominance: -0.55, intensity: 0.82, certainty: -0.4, social: -0.05, description: "Under too much pressure" },
    { id: "E60", name: "Anxious", emoji: "😰", quadrant: "red", x: -1, y: 4, valence: -0.7, arousal: 0.75, dominance: -0.6, intensity: 0.75, certainty: -0.7, social: -0.1, description: "Worried about what might happen" },
    { id: "E61", name: "Frustrated", emoji: "😤", quadrant: "red", x: -5, y: 3, valence: -0.65, arousal: 0.78, dominance: 0.35, intensity: 0.75, certainty: 0.45, social: 0.15, description: "Blocked from what you want to do" },
    { id: "E62", name: "Exasperated", emoji: "🤦", quadrant: "red", x: -4, y: 3, valence: -0.68, arousal: 0.75, dominance: 0.3, intensity: 0.78, certainty: 0.4, social: 0.25, description: "At your wit's end, fed up" },
    { id: "E63", name: "Aggravated", emoji: "💢", quadrant: "red", x: -3, y: 3, valence: -0.62, arousal: 0.7, dominance: 0.48, intensity: 0.72, certainty: 0.5, social: 0.3, description: "Annoyance building up" },
    { id: "E64", name: "Tense", emoji: "😣", quadrant: "red", x: -2, y: 3, valence: -0.6, arousal: 0.72, dominance: -0.48, intensity: 0.7, certainty: -0.35, social: -0.08, description: "Body and mind wound up tight" },
    { id: "E65", name: "Worried", emoji: "😟", quadrant: "red", x: -1, y: 3, valence: -0.62, arousal: 0.65, dominance: -0.52, intensity: 0.68, certainty: -0.65, social: 0.2, description: "Concerned about something" },
    { id: "E66", name: "Irritated", emoji: "😒", quadrant: "red", x: -5, y: 2, valence: -0.55, arousal: 0.6, dominance: 0.45, intensity: 0.6, certainty: 0.55, social: 0.25, description: "Something's getting on your nerves" },
    { id: "E67", name: "Annoyed", emoji: "🙄", quadrant: "red", x: -4, y: 2, valence: -0.5, arousal: 0.55, dominance: 0.42, intensity: 0.55, certainty: 0.52, social: 0.22, description: "Bothered by something small" },
    { id: "E68", name: "Pressured", emoji: "🏋️", quadrant: "red", x: -3, y: 2, valence: -0.58, arousal: 0.68, dominance: -0.42, intensity: 0.68, certainty: -0.3, social: 0.15, description: "Feeling pushed by deadlines or demands" },
    { id: "E69", name: "Nervous", emoji: "😬", quadrant: "red", x: -2, y: 2, valence: -0.58, arousal: 0.68, dominance: -0.55, intensity: 0.65, certainty: -0.55, social: 0.05, description: "Butterflies in your stomach" },
    { id: "E70", name: "Uneasy", emoji: "🫤", quadrant: "red", x: -1, y: 2, valence: -0.52, arousal: 0.55, dominance: -0.45, intensity: 0.58, certainty: -0.5, social: 0.08, description: "Something feels off" },
    { id: "E71", name: "Jealous", emoji: "💚", quadrant: "red", x: -5, y: 1, valence: -0.65, arousal: 0.62, dominance: -0.4, intensity: 0.72, certainty: 0.3, social: 0.75, description: "Wanting what someone else has" },
    { id: "E72", name: "Envious", emoji: "👀", quadrant: "red", x: -4, y: 1, valence: -0.6, arousal: 0.55, dominance: -0.35, intensity: 0.68, certainty: 0.25, social: 0.78, description: "Wishing you had their success" },
    { id: "E73", name: "Resentful", emoji: "😾", quadrant: "red", x: -3, y: 1, valence: -0.68, arousal: 0.58, dominance: 0.4, intensity: 0.75, certainty: 0.55, social: 0.6, description: "Holding onto past hurts" },
    { id: "E74", name: "Restless", emoji: "🦶", quadrant: "red", x: -2, y: 1, valence: -0.4, arousal: 0.72, dominance: -0.35, intensity: 0.6, certainty: -0.3, social: -0.05, description: "Can't sit still or settle down" },
    { id: "E75", name: "Impatient", emoji: "⏰", quadrant: "red", x: -1, y: 1, valence: -0.48, arousal: 0.68, dominance: 0.3, intensity: 0.62, certainty: 0.4, social: 0.2, description: "Wanting things to happen faster" },

    // Blue Quadrant (Low Energy Negative) - x: -5 to -1, y: -1 to -5
    { id: "E76", name: "Disappointed", emoji: "😕", quadrant: "blue", x: -5, y: -1, valence: -0.65, arousal: -0.42, dominance: -0.38, intensity: 0.68, certainty: 0.4, social: 0.25, description: "Things didn't turn out as hoped" },
    { id: "E77", name: "Discouraged", emoji: "📉", quadrant: "blue", x: -4, y: -1, valence: -0.6, arousal: -0.48, dominance: -0.52, intensity: 0.65, certainty: -0.25, social: 0.1, description: "Lost confidence, want to give up" },
    { id: "E78", name: "Down", emoji: "😔", quadrant: "blue", x: -3, y: -1, valence: -0.68, arousal: -0.58, dominance: -0.45, intensity: 0.68, certainty: -0.35, social: 0.05, description: "Low mood, not yourself today" },
    { id: "E79", name: "Unhappy", emoji: "🙁", quadrant: "blue", x: -2, y: -1, valence: -0.65, arousal: -0.5, dominance: -0.42, intensity: 0.62, certainty: -0.3, social: 0.08, description: "Things aren't going well" },
    { id: "E80", name: "Embarrassed", emoji: "🫣", quadrant: "blue", x: -1, y: -1, valence: -0.58, arousal: -0.25, dominance: -0.55, intensity: 0.65, certainty: 0.35, social: 0.75, description: "Awkward moment, want to hide" },
    { id: "E81", name: "Guilty", emoji: "😖", quadrant: "blue", x: -5, y: -2, valence: -0.72, arousal: -0.38, dominance: -0.65, intensity: 0.78, certainty: 0.6, social: 0.55, description: "Did something wrong, feel bad about it" },
    { id: "E82", name: "Ashamed", emoji: "😳", quadrant: "blue", x: -4, y: -2, valence: -0.82, arousal: -0.42, dominance: -0.75, intensity: 0.85, certainty: 0.55, social: 0.6, description: "Deeply embarrassed about yourself" },
    { id: "E83", name: "Remorseful", emoji: "🥺", quadrant: "blue", x: -3, y: -2, valence: -0.75, arousal: -0.45, dominance: -0.68, intensity: 0.8, certainty: 0.58, social: 0.52, description: "Wishing you could take it back" },
    { id: "E84", name: "Sad", emoji: "😢", quadrant: "blue", x: -2, y: -2, valence: -0.75, arousal: -0.55, dominance: -0.5, intensity: 0.75, certainty: -0.4, social: 0.1, description: "Feeling down and unhappy" },
    { id: "E85", name: "Gloomy", emoji: "🌧️", quadrant: "blue", x: -1, y: -2, valence: -0.62, arousal: -0.62, dominance: -0.48, intensity: 0.6, certainty: -0.45, social: -0.1, description: "Dark cloud hanging over you" },
    { id: "E86", name: "Lonely", emoji: "🚶", quadrant: "blue", x: -5, y: -3, valence: -0.72, arousal: -0.52, dominance: -0.58, intensity: 0.75, certainty: -0.35, social: -0.85, description: "Feeling disconnected from others" },
    { id: "E87", name: "Isolated", emoji: "🏝️", quadrant: "blue", x: -4, y: -3, valence: -0.7, arousal: -0.55, dominance: -0.55, intensity: 0.72, certainty: -0.4, social: -0.9, description: "Cut off from everyone" },
    { id: "E88", name: "Alienated", emoji: "👤", quadrant: "blue", x: -3, y: -3, valence: -0.68, arousal: -0.48, dominance: -0.52, intensity: 0.7, certainty: -0.42, social: -0.88, description: "Don't fit in, don't belong" },
    { id: "E89", name: "Heartbroken", emoji: "💔", quadrant: "blue", x: -2, y: -3, valence: -0.88, arousal: -0.65, dominance: -0.8, intensity: 0.92, certainty: -0.7, social: 0.7, description: "Deep pain from loss or rejection" },
    { id: "E90", name: "Devastated", emoji: "🖤", quadrant: "blue", x: -1, y: -3, valence: -0.9, arousal: -0.7, dominance: -0.85, intensity: 0.95, certainty: -0.75, social: 0.25, description: "Completely crushed by something" },
    { id: "E91", name: "Tired", emoji: "😴", quadrant: "blue", x: -5, y: -4, valence: -0.45, arousal: -0.8, dominance: -0.42, intensity: 0.7, certainty: -0.1, social: -0.05, description: "Need rest, running on fumes" },
    { id: "E92", name: "Fatigued", emoji: "😪", quadrant: "blue", x: -4, y: -4, valence: -0.48, arousal: -0.85, dominance: -0.48, intensity: 0.75, certainty: -0.15, social: -0.08, description: "Physically worn out" },
    { id: "E93", name: "Drained", emoji: "🪫", quadrant: "blue", x: -3, y: -4, valence: -0.55, arousal: -0.88, dominance: -0.55, intensity: 0.82, certainty: -0.18, social: -0.12, description: "Energy tank is empty" },
    { id: "E94", name: "Exhausted", emoji: "😩", quadrant: "blue", x: -2, y: -4, valence: -0.58, arousal: -0.92, dominance: -0.58, intensity: 0.85, certainty: -0.2, social: -0.15, description: "Completely drained, nothing left" },
    { id: "E95", name: "Bored", emoji: "😑", quadrant: "blue", x: -1, y: -4, valence: -0.38, arousal: -0.68, dominance: -0.2, intensity: 0.45, certainty: 0.3, social: -0.2, description: "Nothing interesting, need stimulation" },
    { id: "E96", name: "Depressed", emoji: "😞", quadrant: "blue", x: -5, y: -5, valence: -0.88, arousal: -0.82, dominance: -0.78, intensity: 0.88, certainty: -0.7, social: -0.5, description: "Persistent heaviness and sadness" },
    { id: "E97", name: "Hopeless", emoji: "🕳️", quadrant: "blue", x: -4, y: -5, valence: -0.92, arousal: -0.8, dominance: -0.9, intensity: 0.92, certainty: -0.88, social: -0.4, description: "Can't see any way forward" },
    { id: "E98", name: "Despairing", emoji: "😭", quadrant: "blue", x: -3, y: -5, valence: -0.95, arousal: -0.85, dominance: -0.92, intensity: 0.98, certainty: -0.85, social: -0.45, description: "No hope left, in deep pain" },
    { id: "E99", name: "Numb", emoji: "😶‍🌫️", quadrant: "blue", x: -2, y: -5, valence: -0.5, arousal: -0.85, dominance: -0.68, intensity: 0.35, certainty: -0.5, social: -0.55, description: "Can't feel anything at all" },
    { id: "E100", name: "Apathetic", emoji: "😐", quadrant: "blue", x: -1, y: -5, valence: -0.45, arousal: -0.75, dominance: -0.3, intensity: 0.4, certainty: 0.25, social: -0.4, description: "Don't care about anything" }
];

// Get emotions by quadrant
export function getEmotionsByQuadrant(quadrant: Quadrant): Emotion[] {
    return EMOTIONS.filter(e => e.quadrant === quadrant);
}

// Get all quadrants with their emotions
export function getEmotionGrid(): Record<Quadrant, Emotion[]> {
    return {
        yellow: getEmotionsByQuadrant('yellow'),
        green: getEmotionsByQuadrant('green'),
        red: getEmotionsByQuadrant('red'),
        blue: getEmotionsByQuadrant('blue')
    };
}
