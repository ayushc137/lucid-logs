import type { Emotion } from '$lib/api/emotions';
import * as blobs2 from 'blobs/v2';
import { QUADRANT_COLORS, seededRandom } from './emotionData';

// --- Indicator Logic ---
export function getIndicatorDots(
	emotion: Emotion,
): { color: string; label: string }[] {
	const dots: { color: string; label: string }[] = [];
	if (emotion.dominance > 0.7)
		dots.push({ color: '#fbbf24', label: 'Empowered' });
	else if (emotion.dominance < -0.5)
		dots.push({ color: '#a78bfa', label: 'Vulnerable' });
	if (emotion.social > 0.6) dots.push({ color: '#60a5fa', label: 'Social' });
	if (emotion.certainty < -0.3)
		dots.push({ color: '#9ca3af', label: 'Uncertain' });
	if (emotion.intensity > 0.85)
		dots.push({ color: '#f87171', label: 'Intense' });
	return dots;
}

// --- OpenMoji Logic ---
// Check if string looks like a hex code (e.g., "1F9B6" or "1F62E-200D-1F4A8")
function isHexCode(str: string): boolean {
	return /^[0-9A-F]+(-[0-9A-F]+)*$/i.test(str);
}

export function getOpenMojiUrl(emoji: string): string {
	try {
		let hex: string;

		if (isHexCode(emoji)) {
			// Already a hex code from backend, use directly (uppercase)
			hex = emoji.toUpperCase();
		} else {
			// Actual emoji character, convert to hex code points
			hex = Array.from(emoji)
				.map((c) => c.codePointAt(0)?.toString(16).toUpperCase())
				.filter(Boolean)
				.join('-');
		}

		return `https://openmoji.org/data/color/svg/${hex}.svg`;
	} catch (e) {
		console.warn('Failed to generate OpenMoji URL for', emoji, e);
		return '';
	}
}

// --- Color Logic ---
export function getEmotionColors(emotion: Emotion) {
	const quadrant = emotion.quadrant as keyof typeof QUADRANT_COLORS;
	const baseColors = QUADRANT_COLORS[quadrant];
	const absValence = Math.abs(emotion.valence);
	const absArousal = Math.abs(emotion.arousal);
	return {
		...baseColors,
		opacity: 0.75 + absValence * 0.12 + absArousal * 0.13,
	};
}

// --- Blob Generation Logic ---
export const BLOB_SIZE = 70;

export interface BlobLayer {
	path: string;
	altPath1: string;
	altPath2: string;
	rotation: number;
	offsetX: number;
	offsetY: number;
	scale: number;
}

function getLayerCount(emotion: Emotion): number {
	const absArousal = Math.abs(emotion.arousal);
	return Math.max(2, Math.min(4, Math.floor(absArousal * 2.5) + 2));
}

function getComplexity(emotion: Emotion): {
	edges: number;
	randomness: number;
} {
	const pleasantness = (emotion.valence + 1) / 2;
	return {
		edges: Math.floor(3 + (1 - pleasantness) * 7),
		randomness: Math.floor(2 + (1 - pleasantness) * 5),
	};
}

function generateBlobPath(
	seed: string,
	complexity: { edges: number; randomness: number },
): string {
	const rng = seededRandom(seed);
	return blobs2.svgPath({
		seed: rng() * 1000000,
		extraPoints: complexity.edges,
		randomness: complexity.randomness,
		size: BLOB_SIZE,
	});
}

export function generateLayers(emotion: Emotion): BlobLayer[] {
	const rng = seededRandom(emotion.id);
	const count = getLayerCount(emotion);
	const complexity = getComplexity(emotion);
	const layers: BlobLayer[] = [];

	for (let i = 0; i < count; i++) {
		const layerComplexity = {
			edges: complexity.edges + Math.floor(rng() * 2),
			randomness: complexity.randomness + Math.floor(rng() * 2),
		};
		const offsetAmount = 3 + i * 2;
		const baseScale = 0.75 + (i / count) * 0.35;

		layers.push({
			path: generateBlobPath(`${emotion.id}_l${i}`, layerComplexity),
			altPath1: generateBlobPath(`${emotion.id}_l${i}_a1`, layerComplexity),
			altPath2: generateBlobPath(`${emotion.id}_l${i}_a2`, layerComplexity),
			rotation: (rng() - 0.5) * 30,
			offsetX: (rng() - 0.5) * offsetAmount,
			offsetY: (rng() - 0.5) * offsetAmount,
			scale: baseScale + (rng() - 0.5) * 0.1,
		});
	}
	return layers;
}

export function getAnimSpeed(emotion: Emotion): number {
	const absArousal = Math.abs(emotion.arousal);
	return 0.5 + (1 - absArousal) * 1.2;
}
