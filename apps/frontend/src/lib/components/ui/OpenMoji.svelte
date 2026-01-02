<script lang="ts">
    /**
     * OpenMoji Component
     * - Displays emojis using OpenMoji SVG icons
     * - Lazy loading with IntersectionObserver
     * - Fallback to native emoji on error
     * - Caches URLs to avoid re-computation
     */

    interface Props {
        emoji: string;
        alt?: string;
        size?: "xs" | "sm" | "md" | "lg" | "xl";
        class?: string;
    }

    let {
        emoji,
        alt = "",
        size = "md",
        class: className = "",
    }: Props = $props();

    // Cache for computed URLs
    const urlCache = new Map<string, string>();

    let isLoaded = $state(false);
    let hasError = $state(false);
    let isVisible = $state(false);
    let imgElement: HTMLImageElement;

    // Size classes mapping
    const sizeClasses: Record<string, string> = {
        xs: "w-3 h-3",
        sm: "w-4 h-4",
        md: "w-6 h-6",
        lg: "w-8 h-8",
        xl: "w-10 h-10",
    };

    // Check if string looks like a hex code (e.g., "1F9B6" or "1F62E-200D-1F4A8")
    function isHexCode(str: string): boolean {
        return /^[0-9A-F]+(-[0-9A-F]+)*$/i.test(str);
    }

    // Generate OpenMoji URL from emoji (hex code or character)
    function getOpenMojiUrl(emoji: string): string {
        if (urlCache.has(emoji)) {
            return urlCache.get(emoji)!;
        }

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
                    .join("-");
            }

            const url = `https://openmoji.org/data/color/svg/${hex}.svg`;
            urlCache.set(emoji, url);
            return url;
        } catch (e) {
            console.warn("Failed to generate OpenMoji URL for", emoji, e);
            return "";
        }
    }

    // Convert hex code to actual emoji character for fallback
    function hexToEmoji(input: string): string {
        if (!isHexCode(input)) {
            return input; // Already an emoji character
        }
        try {
            return input
                .split("-")
                .map((hex) => String.fromCodePoint(parseInt(hex, 16)))
                .join("");
        } catch {
            return input;
        }
    }

    const url = $derived(getOpenMojiUrl(emoji));
    const sizeClass = $derived(sizeClasses[size] || sizeClasses.md);
    const fallbackEmoji = $derived(hexToEmoji(emoji));

    function handleLoad() {
        isLoaded = true;
    }

    function handleError() {
        hasError = true;
    }

    // Lazy loading with IntersectionObserver
    function lazyLoad(node: HTMLElement) {
        const observer = new IntersectionObserver(
            (entries) => {
                entries.forEach((entry) => {
                    if (entry.isIntersecting) {
                        isVisible = true;
                        observer.unobserve(node);
                    }
                });
            },
            { rootMargin: "50px" },
        );

        observer.observe(node);

        return {
            destroy() {
                observer.disconnect();
            },
        };
    }
</script>

<span
    class="inline-flex items-center justify-center {sizeClass} {className}"
    use:lazyLoad
>
    {#if hasError}
        <!-- Fallback to native emoji -->
        <span class="leading-none">{fallbackEmoji}</span>
    {:else if isVisible}
        <img
            bind:this={imgElement}
            src={url}
            {alt}
            class="{sizeClass} object-contain transition-opacity duration-200"
            class:opacity-0={!isLoaded}
            class:opacity-100={isLoaded}
            onload={handleLoad}
            onerror={handleError}
            loading="lazy"
            decoding="async"
        />
    {:else}
        <!-- Placeholder while not visible -->
        <span class="leading-none opacity-30">{fallbackEmoji}</span>
    {/if}
</span>
