/**
 * Main entry point for shared library exports.
 * Import from '$lib' or from specific submodules like '$lib/api'.
 *
 * @example
 * import { api, getTasks, type Task } from '$lib/api';
 * import { cn } from '$lib/utils';
 */

// Re-export API module for convenience
export * from './api';

// Re-export utilities
export { cn } from './utils';
