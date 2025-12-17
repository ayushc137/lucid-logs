<script lang="ts">
    import { onMount, onDestroy } from "svelte";
    import { Editor } from "@tiptap/core";
    import StarterKit from "@tiptap/starter-kit";
    import Placeholder from "@tiptap/extension-placeholder";
    import Link from "@tiptap/extension-link";
    import TaskList from "@tiptap/extension-task-list";
    import TaskItem from "@tiptap/extension-task-item";
    import {
        Bold,
        Italic,
        Strikethrough,
        Code,
        List,
        ListOrdered,
        Quote,
        Minus,
        Link as LinkIcon,
        ListTodo,
        Undo,
        Redo,
    } from "lucide-svelte";
    import { cn } from "$lib/utils";

    interface Props {
        value?: string;
        placeholder?: string;
        class?: string;
        minHeight?: string;
        showToolbar?: boolean;
        onchange?: (value: string) => void;
    }

    let {
        value = $bindable(""),
        placeholder = "Write something...",
        class: className = "",
        minHeight = "150px",
        showToolbar = true,
        onchange,
    }: Props = $props();

    let element: HTMLElement;
    let editor: Editor | null = null;
    let isFocused = $state(false);

    onMount(() => {
        editor = new Editor({
            element,
            extensions: [
                StarterKit.configure({
                    heading: { levels: [1, 2, 3] },
                }),
                Placeholder.configure({
                    placeholder,
                }),
                Link.configure({
                    openOnClick: false,
                    HTMLAttributes: {
                        class: "text-primary underline cursor-pointer",
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
                    class: "prose prose-sm max-w-none focus:outline-none",
                },
            },
            onUpdate: ({ editor }) => {
                const html = editor.getHTML();
                value = html;
                onchange?.(html);
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

    function setLink() {
        if (!editor) return;
        const previousUrl = editor.getAttributes("link").href;
        const url = window.prompt("URL", previousUrl);

        if (url === null) return;
        if (url === "") {
            editor.chain().focus().extendMarkRange("link").unsetLink().run();
            return;
        }

        editor
            .chain()
            .focus()
            .extendMarkRange("link")
            .setLink({ href: url })
            .run();
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
            isActive: () => editor?.isActive("bold") ?? false,
            title: "Bold",
        },
        {
            icon: Italic,
            action: () => editor?.chain().focus().toggleItalic().run(),
            isActive: () => editor?.isActive("italic") ?? false,
            title: "Italic",
        },
        {
            icon: Strikethrough,
            action: () => editor?.chain().focus().toggleStrike().run(),
            isActive: () => editor?.isActive("strike") ?? false,
            title: "Strikethrough",
        },
        {
            icon: Code,
            action: () => editor?.chain().focus().toggleCode().run(),
            isActive: () => editor?.isActive("code") ?? false,
            title: "Code",
        },
    ];

    const listButtons: ToolbarButton[] = [
        {
            icon: List,
            action: () => editor?.chain().focus().toggleBulletList().run(),
            isActive: () => editor?.isActive("bulletList") ?? false,
            title: "Bullet List",
        },
        {
            icon: ListOrdered,
            action: () => editor?.chain().focus().toggleOrderedList().run(),
            isActive: () => editor?.isActive("orderedList") ?? false,
            title: "Ordered List",
        },
        {
            icon: ListTodo,
            action: () => editor?.chain().focus().toggleTaskList().run(),
            isActive: () => editor?.isActive("taskList") ?? false,
            title: "Task List",
        },
    ];

    const blockButtons: ToolbarButton[] = [
        {
            icon: Quote,
            action: () => editor?.chain().focus().toggleBlockquote().run(),
            isActive: () => editor?.isActive("blockquote") ?? false,
            title: "Quote",
        },
        {
            icon: Minus,
            action: () => editor?.chain().focus().setHorizontalRule().run(),
            isActive: () => false,
            title: "Divider",
        },
        {
            icon: LinkIcon,
            action: setLink,
            isActive: () => editor?.isActive("link") ?? false,
            title: "Link",
        },
    ];
</script>

<div
    class={cn(
        "rich-editor rounded-lg border transition-all duration-200",
        isFocused ? "border-primary ring-2 ring-primary/20" : "border-base-300",
        className,
    )}
>
    {#if showToolbar}
        <div
            class="flex flex-wrap items-center gap-0.5 p-2 border-b border-base-300 bg-base-200/50 rounded-t-lg"
        >
            <!-- Text formatting -->
            <div class="flex items-center gap-0.5">
                {#each toolbarButtons as btn}
                    <button
                        type="button"
                        class={cn(
                            "btn btn-ghost btn-xs btn-square",
                            btn.isActive() && "bg-primary/20 text-primary",
                        )}
                        onclick={btn.action}
                        title={btn.title}
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
                            "btn btn-ghost btn-xs btn-square",
                            btn.isActive() && "bg-primary/20 text-primary",
                        )}
                        onclick={btn.action}
                        title={btn.title}
                    >
                        <btn.icon class="w-3.5 h-3.5" />
                    </button>
                {/each}
            </div>

            <div class="divider divider-horizontal mx-1 h-4"></div>

            <!-- Blocks -->
            <div class="flex items-center gap-0.5">
                {#each blockButtons as btn}
                    <button
                        type="button"
                        class={cn(
                            "btn btn-ghost btn-xs btn-square",
                            btn.isActive() && "bg-primary/20 text-primary",
                        )}
                        onclick={btn.action}
                        title={btn.title}
                    >
                        <btn.icon class="w-3.5 h-3.5" />
                    </button>
                {/each}
            </div>

            <div class="flex-1"></div>

            <!-- Undo/Redo -->
            <div class="flex items-center gap-0.5">
                <button
                    type="button"
                    class="btn btn-ghost btn-xs btn-square"
                    onclick={() => editor?.chain().focus().undo().run()}
                    disabled={!editor?.can().undo()}
                    title="Undo"
                >
                    <Undo class="w-3.5 h-3.5" />
                </button>
                <button
                    type="button"
                    class="btn btn-ghost btn-xs btn-square"
                    onclick={() => editor?.chain().focus().redo().run()}
                    disabled={!editor?.can().redo()}
                    title="Redo"
                >
                    <Redo class="w-3.5 h-3.5" />
                </button>
            </div>
        </div>
    {/if}

    <div
        bind:this={element}
        class="p-3 overflow-y-auto"
        style="min-height: {minHeight};"
    ></div>
</div>

<style>
    :global(.rich-editor .ProseMirror) {
        min-height: inherit;
    }

    :global(.rich-editor .ProseMirror p) {
        margin: 0.5em 0;
    }

    :global(.rich-editor .ProseMirror:first-child) {
        margin-top: 0;
    }

    :global(.rich-editor .ProseMirror > *:last-child) {
        margin-bottom: 0;
    }

    :global(.rich-editor .ProseMirror h1) {
        font-size: 1.5em;
        font-weight: 700;
        margin: 0.5em 0;
    }

    :global(.rich-editor .ProseMirror h2) {
        font-size: 1.25em;
        font-weight: 600;
        margin: 0.5em 0;
    }

    :global(.rich-editor .ProseMirror h3) {
        font-size: 1.1em;
        font-weight: 600;
        margin: 0.5em 0;
    }

    :global(.rich-editor .ProseMirror ul[data-type="taskList"]) {
        list-style: none;
        padding: 0;
    }

    :global(.rich-editor .ProseMirror ul[data-type="taskList"] li) {
        display: flex;
        align-items: flex-start;
        gap: 0.5rem;
    }

    :global(.rich-editor .ProseMirror ul[data-type="taskList"] li > label) {
        flex-shrink: 0;
        user-select: none;
    }

    :global(
            .rich-editor
                .ProseMirror
                ul[data-type="taskList"]
                li
                > label
                input[type="checkbox"]
        ) {
        cursor: pointer;
    }

    :global(.rich-editor .ProseMirror ul[data-type="taskList"] li > div) {
        flex: 1;
    }

    :global(
            .rich-editor
                .ProseMirror
                ul[data-type="taskList"]
                li[data-checked="true"]
                > div
        ) {
        text-decoration: line-through;
        opacity: 0.6;
    }
</style>
