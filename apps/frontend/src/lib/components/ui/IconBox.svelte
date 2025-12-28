<script lang="ts">
    import type { Snippet } from "svelte";

    type Size = "sm" | "md" | "lg" | "xl";
    type Variant = "solid" | "subtle";

    interface Props {
        /** Size of the icon box */
        size?: Size;
        /** Color variant */
        variant?: Variant;
        /** Semantic color (DaisyUI) */
        color?:
            | "primary"
            | "secondary"
            | "accent"
            | "info"
            | "success"
            | "warning"
            | "error";
        /** Custom class */
        class?: string;
        /** Icon slot */
        children: Snippet;
    }

    let {
        size = "md",
        variant = "solid",
        color = "primary",
        class: className = "",
        children,
    }: Props = $props();

    const sizeClasses: Record<Size, string> = {
        sm: "w-9 h-9 rounded-lg",
        md: "w-10 h-10 rounded-xl",
        lg: "w-16 h-16 rounded-full",
        xl: "w-20 h-20 rounded-full",
    };

    const solidClasses: Record<string, string> = {
        primary: "bg-primary text-primary-content",
        secondary: "bg-secondary text-secondary-content",
        accent: "bg-accent text-accent-content",
        info: "bg-info text-info-content",
        success: "bg-success text-success-content",
        warning: "bg-warning text-warning-content",
        error: "bg-error text-error-content",
    };

    const subtleClasses: Record<string, string> = {
        primary: "bg-primary/10 text-primary",
        secondary: "bg-secondary/10 text-secondary",
        accent: "bg-accent/10 text-accent",
        info: "bg-info/10 text-info",
        success: "bg-success/10 text-success",
        warning: "bg-warning/10 text-warning",
        error: "bg-error/10 text-error",
    };

    const colorClass = $derived(
        variant === "solid" ? solidClasses[color] : subtleClasses[color],
    );
</script>

<div
    class="flex items-center justify-center {sizeClasses[
        size
    ]} {colorClass} {className}"
>
    {@render children()}
</div>
