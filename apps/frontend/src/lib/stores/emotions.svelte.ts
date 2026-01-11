import { api, unwrap } from '$lib/api/client';
import type { Emotion, EmotionGridResponse } from '$lib/api/emotions';

class EmotionStore {
	// State using runes
	grid = $state<EmotionGridResponse | null>(null);
	byId = $state<Map<string, Emotion>>(new Map());
	all = $state<Emotion[]>([]);
	isLoading = $state(false);
	error = $state<string | null>(null);
	isInitialized = $state(false);

	async init() {
		if (this.isInitialized || this.isLoading) return;

		this.isLoading = true;
		this.error = null;

		try {
			const response = await unwrap<EmotionGridResponse>(
				api.get('emotions/grid'),
			);

			this.grid = response;

			// Flatten grid to array
			const allEmotions = [
				...response.yellow,
				...response.green,
				...response.red,
				...response.blue,
			];

			this.all = allEmotions;

			// Map for O(1) lookup
			const map = new Map<string, Emotion>();
			allEmotions.forEach((e) => map.set(e.id, e));
			this.byId = map;

			this.isInitialized = true;
		} catch (e) {
			console.error('Failed to load emotions grid:', e);
			this.error =
				e instanceof Error ? e.message : 'Unknown error loading emotions';
		} finally {
			this.isLoading = false;
		}
	}

	get(id: string): Emotion | undefined {
		return this.byId.get(id);
	}
}

export const emotionStore = new EmotionStore();
