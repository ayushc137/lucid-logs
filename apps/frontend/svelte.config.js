import adapterNode from '@sveltejs/adapter-node';
import adapterStatic from '@sveltejs/adapter-static';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';

// All-in-one / static hosting mode: build a pure client-side SPA that the Go
// backend (or any static file server) can serve directly. Trigger with VITE_SPA=1.
const isSPA = process.env.VITE_SPA === '1';

/** @type {import('@sveltejs/kit').Config} */
const config = {
	// Consult https://svelte.dev/docs/kit/integrations
	// for more information about preprocessors
	preprocess: vitePreprocess(),

	compilerOptions: {
		warningFilter: (warning) =>
			// role="link" on a clickable card row is intentional a11y pattern;
			// svelte-check only whitelists 'button' here.
			warning.code !== 'a11y_no_noninteractive_element_to_interactive_role'
	},

	kit: {
		// adapter-auto only supports some environments, see https://svelte.dev/docs/kit/adapter-auto for a list.
		// If your environment is not supported, or you settled on a specific environment, switch out the adapter.
		// See https://svelte.dev/docs/kit/adapters for more information about adapters.
		adapter: isSPA
			? adapterStatic({ fallback: 'index.html' })
			: adapterNode(),
		prerender: {
			// Dynamic [id] routes aren't crawlable; in SPA mode they render
			// client-side, so it's safe to ignore them during prerender.
			handleUnseenRoutes: 'ignore'
		}
	}
};

export default config;
