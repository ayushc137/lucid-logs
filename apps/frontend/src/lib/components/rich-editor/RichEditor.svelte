<script lang="ts">
import { cn } from '$lib/utils';
import { Editor } from '@tiptap/core';
import Link from '@tiptap/extension-link';
import Placeholder from '@tiptap/extension-placeholder';
import TaskItem from '@tiptap/extension-task-item';
import TaskList from '@tiptap/extension-task-list';
import StarterKit from '@tiptap/starter-kit';
import {
	Bold,
	ChevronDown,
	Code,
	Heading1,
	Heading2,
	Heading3,
	Italic,
	Link as LinkIcon,
	List,
	ListOrdered,
	ListTodo,
	Minus,
	Quote,
	Redo,
	RemoveFormatting,
	Strikethrough,
	Undo,
} from 'lucide-svelte';
import { onDestroy, onMount } from 'svelte';

interface Props {
	value?: string;
	placeholder?: string;
	class?: string;
	minHeight?: string;
	showToolbar?: boolean;
	showFooter?: boolean;
	onchange?: (value: string) => void;
}

let {
	value = $bindable(''),
	placeholder = 'Write something...',
	class: className = '',
	minHeight = '150px',
	showToolbar = true,
	showFooter = false,
	onchange,
}: Props = $props();

let element: HTMLElement;
let editor = $state<Editor | null>(null);
let isFocused = $state(false);
let showHeadingMenu = $state(false);
// Used to trigger reactivity when editor selection/content changes
let editorStateVersion = $state(0);
// Link popover state
let showLinkPopover = $state(false);
let linkUrl = $state('');
let linkInputRef: HTMLInputElement | null = $state(null);
let linkPopoverRef: HTMLDivElement | null = $state(null);

onMount(() => {
	editor = new Editor({
		element,
		extensions: [
			StarterKit.configure({
				heading: { levels: [1, 2, 3] },
				// Disable link from StarterKit since we add it separately with custom config
				link: false,
			}),
			Placeholder.configure({
				placeholder,
			}),
			Link.configure({
				openOnClick: false,
				HTMLAttributes: {
					class: 'text-primary underline cursor-pointer',
				},
			}),
			TaskList,
			TaskItem.configure({
				nested: true,
			}),
		],
		content: value,
		editorProps: {
			attributes: {
				class: 'prose prose-sm max-w-none focus:outline-none',
			},
		},
		onUpdate: ({ editor }) => {
			const html = editor.getHTML();
			value = html;
			onchange?.(html);
			editorStateVersion++;
		},
		onSelectionUpdate: () => {
			editorStateVersion++;
		},
		onFocus: () => {
			isFocused = true;
		},
		onBlur: () => {
			isFocused = false;
		},
	});
});

onDestroy(() => {
	editor?.destroy();
});

// Update content when value changes externally
$effect(() => {
	if (editor && value !== editor.getHTML()) {
		editor.commands.setContent(value);
	}
});

function toggleLinkPopover() {
	if (showLinkPopover) {
		cancelLink();
	} else {
		openLinkPopover();
	}
}

function openLinkPopover() {
	if (!editor) return;
	const previousUrl = editor.getAttributes('link').href || '';
	linkUrl = previousUrl;
	showLinkPopover = true;
	// Auto-focus the input after it renders
	requestAnimationFrame(() => {
		linkInputRef?.focus();
		linkInputRef?.select();
	});
}

function applyLink() {
	if (!editor) return;
	if (linkUrl === '') {
		editor.chain().focus().extendMarkRange('link').unsetLink().run();
	} else {
		// Add https:// if no protocol specified
		let url = linkUrl;
		if (url && !url.match(/^https?:\/\//i)) {
			url = `https://${url}`;
		}
		editor.chain().focus().extendMarkRange('link').setLink({ href: url }).run();
	}
	showLinkPopover = false;
	linkUrl = '';
}

function removeLink() {
	if (!editor) return;
	editor.chain().focus().extendMarkRange('link').unsetLink().run();
	showLinkPopover = false;
	linkUrl = '';
}

function cancelLink() {
	showLinkPopover = false;
	linkUrl = '';
	editor?.chain().focus().run();
}

// Handle click outside to close link popover
function handleClickOutside(event: MouseEvent) {
	if (
		showLinkPopover &&
		linkPopoverRef &&
		!linkPopoverRef.contains(event.target as Node)
	) {
		// Check if click was on the link button itself
		const target = event.target as HTMLElement;
		if (!target.closest('[data-link-button]')) {
			cancelLink();
		}
	}
}

// Set up click outside listener
$effect(() => {
	if (showLinkPopover) {
		document.addEventListener('mousedown', handleClickOutside);
		return () => document.removeEventListener('mousedown', handleClickOutside);
	}
});

function setHeading(level: 1 | 2 | 3) {
	if (!editor) return;
	editor.chain().focus().toggleHeading({ level }).run();
	showHeadingMenu = false;
}

function setParagraph() {
	if (!editor) return;
	editor.chain().focus().setParagraph().run();
	showHeadingMenu = false;
}

function clearFormatting() {
	if (!editor) return;
	editor.chain().focus().unsetAllMarks().clearNodes().run();
}

type ToolbarButton = {
	icon: typeof Bold;
	action: () => void;
	isActive: () => boolean;
	title: string;
};

const toolbarButtons: ToolbarButton[] = [
	{
		icon: Bold,
		action: () => editor?.chain().focus().toggleBold().run(),
		isActive: () => editor?.isActive('bold') ?? false,
		title: 'Bold (Ctrl+B)',
	},
	{
		icon: Italic,
		action: () => editor?.chain().focus().toggleItalic().run(),
		isActive: () => editor?.isActive('italic') ?? false,
		title: 'Italic (Ctrl+I)',
	},
	{
		icon: Strikethrough,
		action: () => editor?.chain().focus().toggleStrike().run(),
		isActive: () => editor?.isActive('strike') ?? false,
		title: 'Strikethrough',
	},
	{
		icon: Code,
		action: () => editor?.chain().focus().toggleCode().run(),
		isActive: () => editor?.isActive('code') ?? false,
		title: 'Inline Code',
	},
];

const listButtons: ToolbarButton[] = [
	{
		icon: List,
		action: () => editor?.chain().focus().toggleBulletList().run(),
		isActive: () => editor?.isActive('bulletList') ?? false,
		title: 'Bullet List',
	},
	{
		icon: ListOrdered,
		action: () => editor?.chain().focus().toggleOrderedList().run(),
		isActive: () => editor?.isActive('orderedList') ?? false,
		title: 'Ordered List',
	},
	{
		icon: ListTodo,
		action: () => editor?.chain().focus().toggleTaskList().run(),
		isActive: () => editor?.isActive('taskList') ?? false,
		title: 'Task List',
	},
];

const blockButtons: ToolbarButton[] = [
	{
		icon: Quote,
		action: () => editor?.chain().focus().toggleBlockquote().run(),
		isActive: () => editor?.isActive('blockquote') ?? false,
		title: 'Quote',
	},
	{
		icon: Minus,
		action: () => editor?.chain().focus().setHorizontalRule().run(),
		isActive: () => false,
		title: 'Divider',
	},
	{
		icon: LinkIcon,
		action: toggleLinkPopover,
		isActive: () => editor?.isActive('link') ?? false,
		title: 'Link',
	},
];

// Get current heading level for display
// Using editorStateVersion to trigger reactivity on selection changes
const currentHeadingLabel = $derived.by(() => {
	// This dependency ensures we re-evaluate when editor state changes
	const _ = editorStateVersion;
	if (!editor) return 'Paragraph';
	if (editor.isActive('heading', { level: 1 })) return 'Heading 1';
	if (editor.isActive('heading', { level: 2 })) return 'Heading 2';
	if (editor.isActive('heading', { level: 3 })) return 'Heading 3';
	return 'Paragraph';
});
</script>

<div
    class={cn(
        "rich-editor rounded-lg border transition-all duration-200 flex flex-col",
        isFocused ? "border-primary ring-2 ring-primary/20" : "border-base-300",
        className,
    )}
    style="min-height: {minHeight};"
>
    {#if showToolbar}
        <div
            class="flex flex-wrap items-center gap-0.5 p-2 border-b border-base-300 bg-base-200/50 rounded-t-lg shrink-0"
        >
            <!-- Heading Dropdown -->
            <div class="relative">
                <button
                    type="button"
                    class={cn(
                        "btn btn-ghost btn-xs h-7 gap-1 min-w-[90px] justify-between font-normal",
                        (editor?.isActive("heading", { level: 1 }) ||
                            editor?.isActive("heading", { level: 2 }) ||
                            editor?.isActive("heading", { level: 3 })) &&
                            "bg-primary/20 text-primary",
                    )}
                    onclick={() => (showHeadingMenu = !showHeadingMenu)}
                    onblur={() =>
                        setTimeout(() => (showHeadingMenu = false), 150)}
                >
                    <span class="text-xs">{currentHeadingLabel}</span>
                    <ChevronDown class="w-3 h-3 opacity-50" />
                </button>
                {#if showHeadingMenu}
                    <div
                        class="absolute top-full left-0 mt-1 bg-base-100 border border-base-300 rounded-lg shadow-lg py-1 z-50 min-w-[120px]"
                    >
                        <button
                            type="button"
                            class={cn(
                                "w-full text-left px-3 py-1.5 text-sm hover:bg-base-200 transition-colors",
                                !editor?.isActive("heading") &&
                                    "bg-primary/10 text-primary",
                            )}
                            onmousedown={(e) => {
                                e.preventDefault();
                                setParagraph();
                            }}
                        >
                            Paragraph
                        </button>
                        <button
                            type="button"
                            class={cn(
                                "w-full text-left px-3 py-1.5 text-lg font-bold hover:bg-base-200 transition-colors",
                                editor?.isActive("heading", { level: 1 }) &&
                                    "bg-primary/10 text-primary",
                            )}
                            onmousedown={(e) => {
                                e.preventDefault();
                                setHeading(1);
                            }}
                        >
                            Heading 1
                        </button>
                        <button
                            type="button"
                            class={cn(
                                "w-full text-left px-3 py-1.5 text-base font-bold hover:bg-base-200 transition-colors",
                                editor?.isActive("heading", { level: 2 }) &&
                                    "bg-primary/10 text-primary",
                            )}
                            onmousedown={(e) => {
                                e.preventDefault();
                                setHeading(2);
                            }}
                        >
                            Heading 2
                        </button>
                        <button
                            type="button"
                            class={cn(
                                "w-full text-left px-3 py-1.5 text-sm font-semibold hover:bg-base-200 transition-colors",
                                editor?.isActive("heading", { level: 3 }) &&
                                    "bg-primary/10 text-primary",
                            )}
                            onmousedown={(e) => {
                                e.preventDefault();
                                setHeading(3);
                            }}
                        >
                            Heading 3
                        </button>
                    </div>
                {/if}
            </div>

            <div class="divider divider-horizontal mx-1 h-4"></div>

            <!-- Text formatting -->
            <div class="flex items-center gap-0.5">
                {#each toolbarButtons as btn}
                    <button
                        type="button"
                        class={cn(
                            "btn btn-ghost btn-xs btn-square tooltip tooltip-bottom",
                            btn.isActive() && "bg-primary/20 text-primary",
                        )}
                        onclick={btn.action}
                        data-tip={btn.title}
                    >
                        <btn.icon class="w-3.5 h-3.5" />
                    </button>
                {/each}
            </div>

            <div class="divider divider-horizontal mx-1 h-4"></div>

            <!-- Lists -->
            <div class="flex items-center gap-0.5">
                {#each listButtons as btn}
                    <button
                        type="button"
                        class={cn(
                            "btn btn-ghost btn-xs btn-square tooltip tooltip-bottom",
                            btn.isActive() && "bg-primary/20 text-primary",
                        )}
                        onclick={btn.action}
                        data-tip={btn.title}
                    >
                        <btn.icon class="w-3.5 h-3.5" />
                    </button>
                {/each}
            </div>

            <div class="divider divider-horizontal mx-1 h-4"></div>

            <!-- Blocks -->
            <div class="flex items-center gap-0.5 relative">
                {#each blockButtons as btn}
                    <button
                        type="button"
                        class={cn(
                            "btn btn-ghost btn-xs btn-square tooltip tooltip-bottom",
                            btn.isActive() && "bg-primary/20 text-primary",
                        )}
                        onclick={btn.action}
                        data-tip={btn.title}
                    >
                        <btn.icon class="w-3.5 h-3.5" />
                    </button>
                {/each}

                <!-- Link Popover -->
                {#if showLinkPopover}
                    <div
                        bind:this={linkPopoverRef}
                        class="absolute top-full right-0 mt-1 bg-base-100 border border-base-300 rounded-lg shadow-xl p-3 z-50 min-w-[300px] animate-in fade-in slide-in-from-top-2 duration-150"
                    >
                        <div
                            class="text-xs font-medium mb-2 opacity-60 flex items-center justify-between"
                        >
                            <span>Insert Link</span>
                            <button
                                type="button"
                                class="btn btn-ghost btn-xs btn-circle opacity-50 hover:opacity-100"
                                onclick={cancelLink}
                            >
                                ✕
                            </button>
                        </div>
                        <input
                            bind:this={linkInputRef}
                            type="url"
                            bind:value={linkUrl}
                            placeholder="https://example.com or paste URL"
                            class="input input-sm input-bordered w-full mb-2 focus:input-primary"
                            onkeydown={(e) => {
                                if (e.key === "Enter") {
                                    e.preventDefault();
                                    applyLink();
                                } else if (e.key === "Escape") {
                                    cancelLink();
                                }
                            }}
                        />
                        <div class="flex gap-2">
                            <button
                                type="button"
                                class="btn btn-primary btn-xs flex-1"
                                onclick={applyLink}
                            >
                                Apply
                            </button>
                            {#if editor?.isActive("link")}
                                <button
                                    type="button"
                                    class="btn btn-error btn-outline btn-xs"
                                    onclick={removeLink}
                                >
                                    Remove
                                </button>
                            {/if}
                        </div>
                        <div class="text-[10px] opacity-40 mt-2 text-center">
                            Press Enter to apply • Escape to cancel
                        </div>
                    </div>
                {/if}
            </div>

            <div class="flex-1"></div>

            <!-- Clear formatting -->
            <button
                type="button"
                class="btn btn-ghost btn-xs btn-square tooltip tooltip-bottom"
                onclick={clearFormatting}
                data-tip="Clear Formatting"
            >
                <RemoveFormatting class="w-3.5 h-3.5" />
            </button>

            <!-- Undo/Redo -->
            <div class="flex items-center gap-0.5">
                <button
                    type="button"
                    class="btn btn-ghost btn-xs btn-square tooltip tooltip-bottom"
                    onclick={() => editor?.chain().focus().undo().run()}
                    disabled={!editor?.can().undo()}
                    data-tip="Undo (Ctrl+Z)"
                >
                    <Undo class="w-3.5 h-3.5" />
                </button>
                <button
                    type="button"
                    class="btn btn-ghost btn-xs btn-square tooltip tooltip-bottom"
                    onclick={() => editor?.chain().focus().redo().run()}
                    disabled={!editor?.can().redo()}
                    data-tip="Redo (Ctrl+Y)"
                >
                    <Redo class="w-3.5 h-3.5" />
                </button>
            </div>
        </div>
    {/if}

    <div
        bind:this={element}
        class="p-3 overflow-auto overflow-x-hidden flex-1"
    ></div>

    {#if showFooter}
        <div
            class="flex items-center justify-between px-3 py-1.5 border-t border-base-300 bg-base-200/30 text-xs opacity-50 rounded-b-lg shrink-0"
        >
            <div class="flex items-center gap-3">
                <span
                    ><kbd class="kbd kbd-xs">Ctrl</kbd>+<kbd class="kbd kbd-xs"
                        >B</kbd
                    > Bold</span
                >
                <span
                    ><kbd class="kbd kbd-xs">Ctrl</kbd>+<kbd class="kbd kbd-xs"
                        >I</kbd
                    > Italic</span
                >
            </div>
            <span>Markdown shortcuts supported</span>
        </div>
    {/if}
</div>
