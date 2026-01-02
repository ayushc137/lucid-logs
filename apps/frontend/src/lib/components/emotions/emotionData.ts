// =============================================================================
// EMOTION DATA & UTILITIES
// UI utilities for emotion grid visualization
// =============================================================================

import type { Emotion } from "$lib/api/emotions";
export type { Emotion };

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
    yellow: { label: 'High Energy Pleasant', energyLabel: 'High', pleasantnessLabel: 'Pleasant' },
    green: { label: 'Low Energy Pleasant', energyLabel: 'Low', pleasantnessLabel: 'Pleasant' },
    red: { label: 'High Energy Unpleasant', energyLabel: 'High', pleasantnessLabel: 'Unpleasant' },
    blue: { label: 'Low Energy Unpleasant', energyLabel: 'Low', pleasantnessLabel: 'Unpleasant' }
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
