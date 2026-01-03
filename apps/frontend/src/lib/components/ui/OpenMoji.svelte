<script context="module" lang="ts">
    // Shared cache for computed URLs to avoid re-computation across instances
    const urlCache = new Map<string, string>();

    // Shared cache for loaded images to avoid fade-in animation for already loaded images
    const loadedImageCache = new Set<string>();

    // Check if string looks like a hex code (e.g., "1F9B6" or "1F62E-200D-1F4A8")
    function isHexCode(str: string): boolean {
        return /^[0-9A-F]+(-[0-9A-F]+)*$/i.test(str);
    }

    // Generate OpenMoji URL from emoji (hex code or character)
    export function getOpenMojiUrl(emoji: string): string {
        if (!emoji) return "";

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
        if (!input) return "";
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
</script>

<script lang="ts">
    /**
     * OpenMoji Component
     * - Displays emojis using OpenMoji SVG icons
     * - Lazy loading with IntersectionObserver
     * - Fallback to native emoji on error
     * - Caches URLs to avoid re-computation
     * - Caches loaded state to prevent redundant animations
     */

    interface Props {
        emoji: string;
        alt?: string;
        size?: "xs" | "sm" | "md" | "lg" | "xl" | number;
        class?: string;
    }

    let {
        emoji,
        alt = "",
        size = "md",
        class: className = "",
    }: Props = $props();

    // Derived values
    const url = $derived(getOpenMojiUrl(emoji));
    // Support both named sizes and numeric pixel sizes
    const isNumericSize = $derived(typeof size === "number");
    const sizeClasses: Record<string, string> = {
        xs: "w-3 h-3",
        sm: "w-4 h-4",
        md: "w-6 h-6",
        lg: "w-8 h-8",
        xl: "w-10 h-10",
    };

    // Optimization: check if image is already loaded in global cache
    // We initialize based on cache presence to avoid initial false state
    let isLoaded = $state(false);
    let hasError = $state(false);
    let isVisible = $state(false);
    let imgElement: HTMLImageElement;

    // React to URL changes - update isLoaded state
    $effect(() => {
        if (loadedImageCache.has(url)) {
            isLoaded = true;
            hasError = false;
        } else {
            // Only reset if it's a new URL not in cache
            if (isLoaded) isLoaded = false;
        }
    });

    const sizeClass = $derived(
        isNumericSize ? "" : sizeClasses[size as string] || sizeClasses.md,
    );
    const sizeStyle = $derived(
        isNumericSize ? `width: ${size}px; height: ${size}px;` : "",
    );
    const fallbackEmoji = $derived(hexToEmoji(emoji));

    function handleLoad() {
        isLoaded = true;
        if (url) loadedImageCache.add(url);
    }

    function handleError() {
        hasError = true;
        // Optionally remove from cache if error?
        // loadedImageCache.delete(url);
    }

    // Lazy loading with IntersectionObserver
    function lazyLoad(node: HTMLElement) {
        // Optimization: If we already loaded this image before (in app session),
        // we might surely be able to show it without observer delay?
        // But for list performance, keeping IntersectionObserver is safer.

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
    style={sizeStyle}
    use:lazyLoad
>
    {#if hasError}
        <!-- Fallback to native emoji -->
        <span
            class="leading-none"
            style={sizeStyle ? `font-size: ${(size as number) * 0.7}px;` : ""}
            >{fallbackEmoji}</span
        >
    {:else if isVisible}
        <img
            bind:this={imgElement}
            src={url}
            {alt}
            class="{sizeClass} object-contain transition-opacity duration-200"
            style={sizeStyle}
            class:opacity-0={!isLoaded}
            class:opacity-100={isLoaded}
            onload={handleLoad}
            onerror={handleError}
            loading="lazy"
            decoding="async"
        />
    {:else}
        <!-- Placeholder while not visible -->
        <span
            class="leading-none opacity-30"
            style={sizeStyle ? `font-size: ${(size as number) * 0.7}px;` : ""}
            >{fallbackEmoji}</span
        >
    {/if}
</span>
