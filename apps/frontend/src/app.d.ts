// See https://svelte.dev/docs/kit/types#app.d.ts
// for information about these interfaces
declare global {
	namespace App {
		// interface Error {}
		// interface Locals {}
		// interface PageData {}
		// interface PageState {}
		// interface Platform {}
	}

	// emoji-picker-element custom element
	namespace svelteHTML {
		interface IntrinsicElements {
			'emoji-picker': {
				class?: string;
				'onemoji-click'?: (
					event: CustomEvent<{
						unicode: string;
						emoji: { annotation: string };
					}>,
				) => void;
			};
		}
	}
}

export {};
