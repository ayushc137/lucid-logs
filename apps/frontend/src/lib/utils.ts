import { type ClassValue, clsx } from 'clsx';
import { twMerge } from 'tailwind-merge';

export function cn(...inputs: ClassValue[]) {
	return twMerge(clsx(inputs));
}

/**
 * Strip HTML tags from rich text content while preserving formatting intent.
 * Handles paragraphs, line breaks, lists, etc. for clean plain text display.
 *
 * @param html - HTML string from rich text editor
 * @param options - Configuration options
 * @returns Clean plain text with preserved formatting
 */
export function stripHtml(
	html: string | null | undefined,
	options?: {
		/** Maximum length of output (truncates with ...) */
		maxLength?: number;
		/** Replace newlines with a specific separator (default: keep newlines) */
		newlineSeparator?: string;
		/** Collapse multiple spaces/newlines */
		compact?: boolean;
	},
): string {
	if (!html) return '';

	let text = html;

	// Handle task lists (checkboxes) - convert to bullet points
	text = text.replace(
		/<li[^>]*data-checked="true"[^>]*>([^<]*)<\/li>/gi,
		'✓ $1\n',
	);
	text = text.replace(
		/<li[^>]*data-checked="false"[^>]*>([^<]*)<\/li>/gi,
		'○ $1\n',
	);

	// Handle ordered list items - try to extract number or use bullet
	text = text.replace(/<li[^>]*>/gi, '• ');
	text = text.replace(/<\/li>/gi, '\n');

	// Handle block elements - add line breaks
	text = text.replace(/<\/p>/gi, '\n');
	text = text.replace(/<p[^>]*>/gi, '');
	text = text.replace(/<br\s*\/?>/gi, '\n');
	text = text.replace(/<\/div>/gi, '\n');
	text = text.replace(/<div[^>]*>/gi, '');
	text = text.replace(/<\/h[1-6]>/gi, '\n');
	text = text.replace(/<h[1-6][^>]*>/gi, '');
	text = text.replace(/<\/blockquote>/gi, '\n');
	text = text.replace(/<blockquote[^>]*>/gi, '');

	// Handle lists wrapper
	text = text.replace(/<\/?[ou]l[^>]*>/gi, '\n');

	// Remove all remaining HTML tags
	text = text.replace(/<[^>]*>/g, '');

	// Decode HTML entities
	text = text.replace(/&nbsp;/gi, ' ');
	text = text.replace(/&amp;/gi, '&');
	text = text.replace(/&lt;/gi, '<');
	text = text.replace(/&gt;/gi, '>');
	text = text.replace(/&quot;/gi, '"');
	text = text.replace(/&#39;/gi, "'");
	text = text.replace(/&hellip;/gi, '…');
	text = text.replace(/&mdash;/gi, '—');
	text = text.replace(/&ndash;/gi, '–');

	// Clean up whitespace
	if (options?.compact) {
		// Collapse all whitespace to single spaces
		text = text.replace(/\s+/g, ' ');
	} else {
		// Normalize line breaks - collapse multiple newlines to max 2
		text = text.replace(/\n{3,}/g, '\n\n');
		// Collapse multiple spaces (but keep newlines)
		text = text.replace(/[^\S\n]+/g, ' ');
	}

	// Trim
	text = text.trim();

	// Apply newline separator if specified
	if (options?.newlineSeparator !== undefined) {
		text = text.replace(/\n/g, options.newlineSeparator);
	}

	// Truncate if max length specified
	if (options?.maxLength && text.length > options.maxLength) {
		text = `${text.slice(0, options.maxLength).trim()}…`;
	}

	return text;
}
