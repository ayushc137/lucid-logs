<script lang="ts">
    /**
     * EMOTION GRID
     * - Fetches all emotion data from backend API
     * - OpenMoji font
     * - Tooltip portaled to body (outside modal) via action
     * - panzoom for smooth pan/zoom with wheel support
     * - Tailwind CSS styling
     * - Client-side search (filters loaded emotions)
     */
    import { onMount, onDestroy, tick } from "svelte";
    import panzoom, { type PanZoom } from "panzoom";
    import * as blobs2 from "blobs/v2";
    import { QUADRANT_COLORS, seededRandom } from "./emotionData";
    import { getEmotionGrid, type Emotion } from "$lib/api/emotions";
    import {
        ZoomIn,
        ZoomOut,
        RotateCcw,
        ArrowUp,
        ArrowDown,
        ArrowLeft,
        ArrowRight,
        Search,
        X,
        Loader2,
    } from "lucide-svelte";

    interface Props {
        selectedEmotion?: Emotion | null;
        onSelect?: (emotion: Emotion) => void;
        showIndicators?: boolean;
        onHoveredChange?: (emotion: Emotion | null) => void;
    }

    let {
        selectedEmotion = $bindable(null),
        onSelect,
        showIndicators = $bindable(false),
        onHoveredChange,
    }: Props = $props();

    // API data state
    let allEmotions = $state<Emotion[]>([]);
    let isLoading = $state(true);
    let loadError = $state<string | null>(null);

    // Hover state
    let hoveredEmotion = $state<Emotion | null>(null);
    let tooltipPosition = $state<{ x: number; y: number } | null>(null);
    let isMouseHovering = $state(false);
    let hoverTimeout: ReturnType<typeof setTimeout> | undefined;

    // Panzoom
    let viewportElement: HTMLDivElement;
    let gridElement: HTMLDivElement;
    let pzInstance: PanZoom | null = null;
    let zoomLevel = $state(1);
    let isPzReady = $state(false);

    // Search state - client-side filtering
    let searchQuery = $state("");
    let showSearchDropdown = $state(false);
    let searchInputElement: HTMLInputElement;

    // Computed search results - filters allEmotions client-side
    const searchResults = $derived.by(() => {
        if (!searchQuery.trim()) return [];
        const query = searchQuery.toLowerCase();
        return allEmotions
            .filter(
                (e) =>
                    e.name.toLowerCase().includes(query) ||
                    e.description.toLowerCase().includes(query),
            )
            .slice(0, 20); // Limit to 20 results
    });

    // Center detection for auto-hover
    let centerEmotion = $state<Emotion | null>(null);

    // Computed grid from API data
    const grid = $derived.by(() => {
        const gridArray: (Emotion | null)[][] = Array(10)
            .fill(null)
            .map(() => Array(10).fill(null));

        for (const emotion of allEmotions) {
            const col = emotion.x < 0 ? emotion.x + 5 : emotion.x + 4;
            const row = emotion.y > 0 ? 5 - emotion.y : Math.abs(emotion.y) + 4;
            if (row >= 0 && row < 10 && col >= 0 && col < 10) {
                gridArray[row][col] = emotion;
            }
        }
        return gridArray;
    });

    // Pre-compute blob layers for all emotions
    const emotionLayers = $derived.by(() => {
        const layers = new Map<string, BlobLayer[]>();
        for (const e of allEmotions) {
            layers.set(e.id, generateLayers(e));
        }
        return layers;
    });

    function findCenterEmotion() {
        if (!viewportElement || !gridElement || allEmotions.length === 0)
            return;

        const viewportRect = viewportElement.getBoundingClientRect();
        const centerX = viewportRect.left + viewportRect.width / 2;
        const centerY = viewportRect.top + viewportRect.height / 2;

        const buttons = Array.from(
            gridElement.querySelectorAll(".emotion-btn"),
        );
        let closestEmotion: Emotion | null = null;
        let closestDistance = Infinity;

        for (const btn of buttons) {
            const rect = btn.getBoundingClientRect();
            const btnCenterX = rect.left + rect.width / 2;
            const btnCenterY = rect.top + rect.height / 2;
            const distance = Math.sqrt(
                Math.pow(centerX - btnCenterX, 2) +
                    Math.pow(centerY - btnCenterY, 2),
            );

            const emotionId = btn.getAttribute("data-emotion-id");
            const emotion = allEmotions.find((e) => e.id === emotionId);

            if (emotion && distance < closestDistance) {
                closestEmotion = emotion;
                closestDistance = distance;
            }
        }

        if (closestEmotion && closestDistance < 100) {
            centerEmotion = closestEmotion;
            if (!isMouseHovering) {
                hoveredEmotion = closestEmotion;
                onHoveredChange?.(closestEmotion);
            }
        }
    }

    /**
     * Pan to a specific emotion in the grid.
     */
    function panToEmotion(emotion: Emotion) {
        if (!pzInstance || !viewportElement || !gridElement) return;

        const viewportRect = viewportElement.getBoundingClientRect();
        const gridRect = gridElement.getBoundingClientRect();
        const transform = pzInstance.getTransform();

        // Find the emotion button
        const btn = gridElement.querySelector(
            `[data-emotion-id="${emotion.id}"]`,
        );
        if (!btn) return;

        const btnRect = btn.getBoundingClientRect();

        // Calculate the position relative to the grid (before transform)
        const btnCenterX =
            (btnRect.left + btnRect.width / 2 - gridRect.left) /
            transform.scale;
        const btnCenterY =
            (btnRect.top + btnRect.height / 2 - gridRect.top) / transform.scale;

        // Calculate the viewport center
        const viewportCenterX = viewportRect.width / 2;
        const viewportCenterY = viewportRect.height / 2;

        // Calculate the new pan position to center the emotion
        const newX = viewportCenterX - btnCenterX * transform.scale;
        const newY = viewportCenterY - btnCenterY * transform.scale;

        pzInstance.smoothMoveTo(newX, newY);
    }

    /**
     * Pan to the center of the grid.
     */
    function panToCenter() {
        if (!pzInstance || !viewportElement || !gridElement) return;

        const viewportRect = viewportElement.getBoundingClientRect();
        const gridRect = gridElement.getBoundingClientRect();
        const transform = pzInstance.getTransform();

        // Calculate grid center before transform
        const gridCenterX = gridRect.width / transform.scale / 2;
        const gridCenterY = gridRect.height / transform.scale / 2;

        // Calculate viewport center
        const viewportCenterX = viewportRect.width / 2;
        const viewportCenterY = viewportRect.height / 2;

        // Calculate new pan position
        const newX = viewportCenterX - gridCenterX * transform.scale;
        const newY = viewportCenterY - gridCenterY * transform.scale;

        pzInstance.smoothMoveTo(newX, newY);
    }

    /**
     * Initialize panzoom and pan to selected emotion or center.
     */
    function initializePanPosition() {
        if (!isPzReady) return;

        if (selectedEmotion) {
            panToEmotion(selectedEmotion);
        } else {
            panToCenter();
        }
        setTimeout(findCenterEmotion, 300);
    }

    // Search functions - now client-side, no API calls needed
    function handleSearchInput(e: Event) {
        const value = (e.target as HTMLInputElement).value;
        searchQuery = value;
        showSearchDropdown = value.length > 0;
    }

    function handleSearchResultClick(emotion: Emotion) {
        handleSelect(emotion);
        panToEmotion(emotion);
        searchQuery = "";
        showSearchDropdown = false;
    }

    function clearSearch() {
        searchQuery = "";
        showSearchDropdown = false;
    }

    function handleSearchFocus() {
        if (searchQuery.length > 0 && searchResults.length > 0) {
            showSearchDropdown = true;
        }
    }

    function handleSearchBlur() {
        // Delay to allow click on results
        setTimeout(() => {
            showSearchDropdown = false;
        }, 200);
    }

    function handleSearchKeydown(e: KeyboardEvent) {
        if (e.key === "Escape") {
            clearSearch();
            searchInputElement?.blur();
        }
    }

    // Fetch emotions from API
    async function loadEmotions() {
        isLoading = true;
        loadError = null;

        try {
            const response = await getEmotionGrid();
            // Combine all quadrants into a single array
            allEmotions = [
                ...response.yellow,
                ...response.green,
                ...response.red,
                ...response.blue,
            ];
        } catch (error) {
            console.error("Failed to load emotions:", error);
            loadError =
                error instanceof Error
                    ? error.message
                    : "Failed to load emotions";
        } finally {
            isLoading = false;
        }
    }

    onMount(async () => {
        // Load emotions from API
        await loadEmotions();

        // Initialize panzoom after emotions are loaded
        await tick();

        if (gridElement) {
            pzInstance = panzoom(gridElement, {
                maxZoom: 2.5,
                minZoom: 0.6,
                bounds: true,
                boundsPadding: 0.2,
                smoothScroll: true, // Enable smooth scrolling for trackpad/wheel
            });

            pzInstance.on("zoom", () => {
                zoomLevel = pzInstance?.getTransform().scale ?? 1;
                findCenterEmotion();
            });

            pzInstance.on("pan", () => {
                findCenterEmotion();
            });

            // Mark as ready after a brief delay to allow rendering
            await tick();
            isPzReady = true;
            initializePanPosition();
        }
    });

    onDestroy(() => {
        pzInstance?.dispose();
        if (hoverTimeout) clearTimeout(hoverTimeout);
    });

    // Watch for selectedEmotion changes to pan to selected item
    $effect(() => {
        if (isPzReady && selectedEmotion) {
            // Small delay to ensure DOM is ready
            setTimeout(() => panToEmotion(selectedEmotion), 50);
        }
    });

    function zoomIn() {
        pzInstance?.smoothZoom(0, 0, 1.3);
    }

    function zoomOut() {
        pzInstance?.smoothZoom(0, 0, 0.7);
    }

    function resetView() {
        pzInstance?.moveTo(0, 0);
        pzInstance?.zoomAbs(0, 0, 1);
        zoomLevel = 1;
        setTimeout(() => {
            if (selectedEmotion) {
                panToEmotion(selectedEmotion);
            } else {
                panToCenter();
            }
            findCenterEmotion();
        }, 100);
    }

    function getEmotionColors(emotion: Emotion) {
        const quadrant = emotion.quadrant as keyof typeof QUADRANT_COLORS;
        const baseColors = QUADRANT_COLORS[quadrant];
        const absValence = Math.abs(emotion.valence);
        const absArousal = Math.abs(emotion.arousal);
        return {
            ...baseColors,
            opacity: 0.75 + absValence * 0.12 + absArousal * 0.13,
        };
    }

    // Blob generation
    const BLOB_SIZE = 70;

    interface BlobLayer {
        path: string;
        altPath1: string;
        altPath2: string;
        rotation: number;
        offsetX: number;
        offsetY: number;
        scale: number;
    }

    function getLayerCount(emotion: Emotion): number {
        const absArousal = Math.abs(emotion.arousal);
        return Math.max(2, Math.min(4, Math.floor(absArousal * 2.5) + 2));
    }

    function getComplexity(emotion: Emotion): {
        edges: number;
        randomness: number;
    } {
        const pleasantness = (emotion.valence + 1) / 2;
        return {
            edges: Math.floor(3 + (1 - pleasantness) * 7),
            randomness: Math.floor(2 + (1 - pleasantness) * 5),
        };
    }

    function generateBlobPath(
        seed: string,
        complexity: { edges: number; randomness: number },
    ): string {
        const rng = seededRandom(seed);
        return blobs2.svgPath({
            seed: rng() * 1000000,
            extraPoints: complexity.edges,
            randomness: complexity.randomness,
            size: BLOB_SIZE,
        });
    }

    function generateLayers(emotion: Emotion): BlobLayer[] {
        const rng = seededRandom(emotion.id);
        const count = getLayerCount(emotion);
        const complexity = getComplexity(emotion);
        const layers: BlobLayer[] = [];

        for (let i = 0; i < count; i++) {
            const layerComplexity = {
                edges: complexity.edges + Math.floor(rng() * 2),
                randomness: complexity.randomness + Math.floor(rng() * 2),
            };
            const offsetAmount = 3 + i * 2;
            const baseScale = 0.75 + (i / count) * 0.35;

            layers.push({
                path: generateBlobPath(`${emotion.id}_l${i}`, layerComplexity),
                altPath1: generateBlobPath(
                    `${emotion.id}_l${i}_a1`,
                    layerComplexity,
                ),
                altPath2: generateBlobPath(
                    `${emotion.id}_l${i}_a2`,
                    layerComplexity,
                ),
                rotation: (rng() - 0.5) * 30,
                offsetX: (rng() - 0.5) * offsetAmount,
                offsetY: (rng() - 0.5) * offsetAmount,
                scale: baseScale + (rng() - 0.5) * 0.1,
            });
        }
        return layers;
    }

    function getAnimSpeed(emotion: Emotion): number {
        const absArousal = Math.abs(emotion.arousal);
        return 0.5 + (1 - absArousal) * 1.2;
    }

    function getIndicatorDots(
        emotion: Emotion,
    ): { color: string; label: string }[] {
        const dots: { color: string; label: string }[] = [];
        if (emotion.dominance > 0.7)
            dots.push({ color: "#fbbf24", label: "Empowered" });
        else if (emotion.dominance < -0.5)
            dots.push({ color: "#a78bfa", label: "Vulnerable" });
        if (emotion.social > 0.6)
            dots.push({ color: "#60a5fa", label: "Social" });
        if (emotion.certainty < -0.3)
            dots.push({ color: "#9ca3af", label: "Uncertain" });
        if (emotion.intensity > 0.85)
            dots.push({ color: "#f87171", label: "Intense" });
        return dots;
    }

    function handleSelect(emotion: Emotion) {
        onSelect?.(emotion);
    }

    // Mouse handlers - update tooltip in DOM portal
    function onMouseEnter(e: MouseEvent, emotion: Emotion) {
        if (hoverTimeout) clearTimeout(hoverTimeout);

        isMouseHovering = true;
        hoveredEmotion = emotion;
        onHoveredChange?.(emotion);
        updateTooltipPosition(e);
    }

    function updateTooltipPosition(e: MouseEvent) {
        // Position to the right of cursor, clamped to viewport
        let x = e.clientX + 16;
        let y = e.clientY;

        // Clamp to viewport edges
        const tooltipWidth = 260;
        const tooltipHeight = 120;

        if (x + tooltipWidth > window.innerWidth - 20) {
            x = e.clientX - tooltipWidth - 16;
        }
        if (y + tooltipHeight > window.innerHeight - 20) {
            y = window.innerHeight - tooltipHeight - 20;
        }
        if (y < 20) y = 20;

        tooltipPosition = { x, y };
    }

    function onMouseMove(e: MouseEvent) {
        if (isMouseHovering && hoveredEmotion) {
            updateTooltipPosition(e);
        }
    }

    function onMouseLeave() {
        if (hoverTimeout) clearTimeout(hoverTimeout);

        hoverTimeout = setTimeout(() => {
            isMouseHovering = false;
            tooltipPosition = null;
            if (centerEmotion) {
                hoveredEmotion = centerEmotion;
                onHoveredChange?.(centerEmotion);
            }
        }, 50);
    }

    function portal(node: HTMLElement) {
        document.body.appendChild(node);
        return {
            destroy() {
                if (node.parentNode) {
                    node.parentNode.removeChild(node);
                }
            },
        };
    }
</script>

<div class="flex flex-col gap-2 h-full">
    <!-- Controls -->
    <div class="flex justify-between items-center shrink-0 gap-2">
        <!-- Search -->
        <div class="relative flex-1 max-w-xs">
            <div class="relative">
                <Search
                    size={14}
                    class="absolute left-2.5 top-1/2 -translate-y-1/2 text-base-content/50 pointer-events-none"
                />
                <input
                    bind:this={searchInputElement}
                    type="text"
                    placeholder="Search emotions..."
                    value={searchQuery}
                    oninput={handleSearchInput}
                    onfocus={handleSearchFocus}
                    onblur={handleSearchBlur}
                    onkeydown={handleSearchKeydown}
                    class="input input-xs input-bordered w-full pl-8 pr-7 h-7 text-xs"
                />
                {#if searchQuery}
                    <button
                        class="absolute right-1.5 top-1/2 -translate-y-1/2 p-0.5 rounded-full hover:bg-base-300 transition-colors"
                        onclick={clearSearch}
                        type="button"
                    >
                        <X size={12} class="text-base-content/50" />
                    </button>
                {/if}
            </div>

            <!-- Search Results Dropdown -->
            {#if showSearchDropdown}
                <div
                    class="absolute z-50 top-full mt-1 left-0 right-0 bg-base-100 border border-base-300 rounded-lg shadow-lg max-h-48 overflow-y-auto"
                >
                    {#if searchResults.length === 0}
                        <div
                            class="py-3 px-3 text-xs text-base-content/50 text-center"
                        >
                            No emotions found
                        </div>
                    {:else}
                        {#each searchResults as result}
                            {@const colors = getEmotionColors(result)}
                            <button
                                class="w-full flex items-center gap-2 px-3 py-2 hover:bg-base-200 transition-colors text-left"
                                onclick={() => handleSearchResultClick(result)}
                                type="button"
                            >
                                <span
                                    class="w-6 h-6 rounded-lg flex items-center justify-center text-sm shrink-0"
                                    style="background: {colors.gradient}"
                                >
                                    {result.emoji}
                                </span>
                                <div class="flex-1 min-w-0">
                                    <div class="text-xs font-medium truncate">
                                        {result.name}
                                    </div>
                                    <div
                                        class="text-[0.65rem] text-base-content/50 truncate"
                                    >
                                        {result.description}
                                    </div>
                                </div>
                            </button>
                        {/each}
                    {/if}
                </div>
            {/if}
        </div>

        <!-- Zoom Controls -->
        <div class="flex items-center gap-0.5">
            <button
                class="btn btn-ghost btn-xs btn-square"
                onclick={zoomOut}
                title="Zoom out"
            >
                <ZoomOut size={14} />
            </button>
            <span class="text-[0.65rem] opacity-60 min-w-8 text-center"
                >{Math.round(zoomLevel * 100)}%</span
            >
            <button
                class="btn btn-ghost btn-xs btn-square"
                onclick={zoomIn}
                title="Zoom in"
            >
                <ZoomIn size={14} />
            </button>
            <button
                class="btn btn-ghost btn-xs btn-square"
                onclick={resetView}
                title="Reset view"
            >
                <RotateCcw size={14} />
            </button>
        </div>

        <label class="label cursor-pointer gap-2">
            <span class="label-text text-xs opacity-70">Details</span>
            <input
                type="checkbox"
                class="toggle toggle-primary toggle-xs"
                bind:checked={showIndicators}
            />
        </label>
    </div>

    <!-- Loading State -->
    {#if isLoading}
        <div class="flex-1 flex items-center justify-center">
            <div class="flex flex-col items-center gap-3">
                <Loader2 size={32} class="animate-spin text-primary" />
                <span class="text-sm text-base-content/60"
                    >Loading emotions...</span
                >
            </div>
        </div>
    {:else if loadError}
        <div class="flex-1 flex items-center justify-center">
            <div class="flex flex-col items-center gap-3 text-center px-4">
                <div class="text-error text-lg">⚠️</div>
                <span class="text-sm text-base-content/70">{loadError}</span>
                <button class="btn btn-sm btn-primary" onclick={loadEmotions}>
                    Retry
                </button>
            </div>
        </div>
    {:else}
        <!-- Panzoom viewport -->
        <div
            class="flex-1 overflow-hidden rounded-2xl bg-gradient-to-br from-base-200/40 to-base-300/30 cursor-grab active:cursor-grabbing relative"
            bind:this={viewportElement}
        >
            <!-- Floating Axis Labels -->
            <div class="absolute inset-0 pointer-events-none z-10">
                <!-- Top: High Energy -->
                <div
                    class="absolute top-3 left-1/2 -translate-x-1/2 flex items-center gap-1 px-3 py-1.5 rounded-full bg-base-100/90 backdrop-blur-sm shadow-sm border border-base-300/50"
                >
                    <ArrowUp size={12} class="text-warning" strokeWidth={2.5} />
                    <span
                        class="text-[0.65rem] font-semibold uppercase tracking-wider text-warning"
                        >High Energy</span
                    >
                </div>
                <!-- Bottom: Low Energy -->
                <div
                    class="absolute bottom-3 left-1/2 -translate-x-1/2 flex items-center gap-1 px-3 py-1.5 rounded-full bg-base-100/90 backdrop-blur-sm shadow-sm border border-base-300/50"
                >
                    <ArrowDown size={12} class="text-info" strokeWidth={2.5} />
                    <span
                        class="text-[0.65rem] font-semibold uppercase tracking-wider text-info"
                        >Low Energy</span
                    >
                </div>
                <!-- Left: Unpleasant (vertical) -->
                <div
                    class="absolute left-3 top-1/2 -translate-y-1/2 flex flex-col items-center gap-1 px-1.5 py-3 rounded-full bg-base-100/90 backdrop-blur-sm shadow-sm border border-base-300/50"
                >
                    <ArrowLeft size={12} class="text-error" strokeWidth={2.5} />
                    <span
                        class="text-[0.6rem] font-semibold uppercase tracking-wider text-error"
                        style="writing-mode: vertical-rl; transform: rotate(180deg);"
                        >Unpleasant</span
                    >
                </div>
                <!-- Right: Pleasant (vertical) -->
                <div
                    class="absolute right-3 top-1/2 -translate-y-1/2 flex flex-col items-center gap-1 px-1.5 py-3 rounded-full bg-base-100/90 backdrop-blur-sm shadow-sm border border-base-300/50"
                >
                    <ArrowRight
                        size={12}
                        class="text-success"
                        strokeWidth={2.5}
                    />
                    <span
                        class="text-[0.6rem] font-semibold uppercase tracking-wider text-success"
                        style="writing-mode: vertical-rl;">Pleasant</span
                    >
                </div>
            </div>

            <div
                class="grid grid-cols-10 gap-3.5 p-5 md:gap-4 md:p-6 lg:gap-[18px] lg:p-7 relative"
                bind:this={gridElement}
            >
                <!-- Grid Lines (Quadrant Dividers) -->
                <div class="absolute inset-0 pointer-events-none z-[1]">
                    <!-- Vertical center line -->
                    <div
                        class="absolute left-1/2 top-4 bottom-4 w-px bg-gradient-to-b from-transparent via-base-content/20 to-transparent"
                    ></div>
                    <!-- Horizontal center line -->
                    <div
                        class="absolute top-1/2 left-4 right-4 h-px bg-gradient-to-r from-transparent via-base-content/20 to-transparent"
                    ></div>
                </div>

                {#each grid as row}
                    {#each row as emotion}
                        <div class="aspect-square relative z-[2]">
                            {#if emotion}
                                {@const isHovered =
                                    hoveredEmotion?.id === emotion.id}
                                {@const isSelected =
                                    selectedEmotion?.id === emotion.id}
                                {@const colors = getEmotionColors(emotion)}
                                {@const layers =
                                    emotionLayers.get(emotion.id) || []}
                                {@const animSpeed = getAnimSpeed(emotion)}
                                {@const layerCount = layers.length}

                                <button
                                    class="emotion-btn relative w-full h-full border-none bg-transparent cursor-pointer flex items-center justify-center rounded-xl overflow-visible outline-none transition-all duration-200"
                                    class:selected={isSelected}
                                    class:auto-hovered={isHovered &&
                                        !isMouseHovering}
                                    data-emotion-id={emotion.id}
                                    onclick={() => handleSelect(emotion)}
                                    onmouseenter={(e) =>
                                        onMouseEnter(e, emotion)}
                                    onmousemove={onMouseMove}
                                    onmouseleave={onMouseLeave}
                                    style="--glow: {colors.glow}; --speed: {animSpeed}s; --primary: {colors.primary};"
                                >
                                    <!-- Selected state ring -->
                                    {#if isSelected}
                                        <div
                                            class="absolute -inset-1.5 rounded-2xl border-2 border-primary animate-pulse pointer-events-none z-[5]"
                                        ></div>
                                        <div
                                            class="absolute -inset-2.5 rounded-2xl bg-primary/10 pointer-events-none z-[4]"
                                        ></div>
                                    {/if}

                                    <div
                                        class="absolute -inset-[12%] w-[124%] h-[124%] pointer-events-none"
                                    >
                                        {#each layers as layer, li}
                                            {@const baseOpacity =
                                                0.45 +
                                                ((layerCount - li) /
                                                    layerCount) *
                                                    0.5}
                                            {@const finalOpacity =
                                                baseOpacity * colors.opacity}
                                            {@const delay = li * 0.1}
                                            <svg
                                                class="blob-svg absolute inset-0 w-full h-full overflow-visible transition-opacity duration-300"
                                                viewBox="0 0 {BLOB_SIZE} {BLOB_SIZE}"
                                                style="
                                                    --rot: {layer.rotation}deg;
                                                    --opacity: {finalOpacity};
                                                    --delay: {delay}s;
                                                    --ox: {layer.offsetX}%;
                                                    --oy: {layer.offsetY}%;
                                                    --scale: {layer.scale};
                                                "
                                            >
                                                <defs>
                                                    <linearGradient
                                                        id="grad-{emotion.id}-{li}"
                                                        x1="0%"
                                                        y1="0%"
                                                        x2="100%"
                                                        y2="100%"
                                                    >
                                                        <stop
                                                            offset="0%"
                                                            stop-color={colors.accent}
                                                        />
                                                        <stop
                                                            offset="50%"
                                                            stop-color={colors.primary}
                                                        />
                                                        <stop
                                                            offset="100%"
                                                            stop-color={colors.secondary}
                                                        />
                                                    </linearGradient>
                                                </defs>
                                                <path
                                                    class="blob-path"
                                                    d={layer.path}
                                                    fill="url(#grad-{emotion.id}-{li})"
                                                >
                                                    {#if isHovered}
                                                        <animate
                                                            attributeName="d"
                                                            dur="{animSpeed}s"
                                                            repeatCount="indefinite"
                                                            values="{layer.path};{layer.altPath1};{layer.altPath2};{layer.path}"
                                                            calcMode="spline"
                                                            keySplines="0.4 0 0.2 1; 0.4 0 0.2 1; 0.4 0 0.2 1"
                                                            begin="{delay}s"
                                                        />
                                                    {/if}
                                                </path>
                                            </svg>
                                        {/each}
                                    </div>

                                    <div
                                        class="relative z-[2] flex flex-col items-center gap-px pointer-events-none"
                                    >
                                        <span
                                            class="emoji font-['OpenMojiColor','Segoe_UI_Emoji','Apple_Color_Emoji',sans-serif] text-[clamp(1rem,2.2vw,1.5rem)] leading-none transition-transform duration-200"
                                            class:scale-110={isSelected}
                                            >{emotion.emoji}</span
                                        >
                                        <span
                                            class="name text-[clamp(0.5rem,1vw,0.7rem)] font-semibold text-white uppercase max-w-full overflow-hidden text-ellipsis whitespace-nowrap hidden sm:inline"
                                            style="text-shadow: 0 1px 2px rgba(0,0,0,0.7);"
                                            >{emotion.name}</span
                                        >
                                    </div>
                                </button>
                            {/if}
                        </div>
                    {/each}
                {/each}
            </div>
        </div>
    {/if}
</div>

<!-- Tooltip - Portaled to body via {#if} to avoid modal clipping -->
{#if hoveredEmotion && tooltipPosition && isMouseHovering && typeof document !== "undefined"}
    {@const colors = getEmotionColors(hoveredEmotion)}
    {@const dots = getIndicatorDots(hoveredEmotion)}
    <div
        class="fixed z-[100000] min-w-[180px] max-w-[260px] bg-base-100 text-base-content rounded-xl shadow-[0_8px_32px_rgba(0,0,0,0.25)] border border-base-300 overflow-hidden pointer-events-none animate-[tooltipIn_0.12s_ease-out]"
        style="left: {tooltipPosition.x}px; top: {tooltipPosition.y}px;"
        use:portal
    >
        <div
            class="flex items-center gap-2.5 px-3 py-2.5 text-white"
            style="background: {colors.gradient};"
        >
            <span
                class="font-['OpenMojiColor','Segoe_UI_Emoji','Apple_Color_Emoji',sans-serif] text-[1.4rem] leading-none"
                >{hoveredEmotion.emoji}</span
            >
            <strong class="text-sm">{hoveredEmotion.name}</strong>
        </div>
        <div class="px-3 py-2.5">
            <p class="m-0 text-xs text-base-content/70 leading-relaxed">
                {hoveredEmotion.description}
            </p>
            {#if showIndicators && dots.length > 0}
                <div class="flex gap-1 mt-2 flex-wrap">
                    {#each dots as d}
                        <span
                            class="text-[0.58rem] px-1.5 py-0.5 rounded-lg font-medium"
                            style="background: color-mix(in srgb, {d.color} 15%, transparent); color: color-mix(in srgb, {d.color} 80%, var(--bc, #333));"
                            >{d.label}</span
                        >
                    {/each}
                </div>
            {/if}
        </div>
    </div>
{/if}

<style>
    /* Minimal custom CSS for complex hover states and SVG transforms that can't be expressed in Tailwind */
    @keyframes tooltipIn {
        from {
            opacity: 0;
            transform: scale(0.95);
        }
        to {
            opacity: 1;
            transform: scale(1);
        }
    }

    @keyframes selectedGlow {
        0%,
        100% {
            filter: drop-shadow(0 0 12px var(--glow))
                drop-shadow(0 0 24px var(--glow));
        }
        50% {
            filter: drop-shadow(0 0 18px var(--glow))
                drop-shadow(0 0 32px var(--glow));
        }
    }

    @keyframes hoverPulse {
        0%,
        100% {
            transform: scale(1.15);
            filter: drop-shadow(0 0 12px var(--glow))
                drop-shadow(0 0 24px var(--glow));
        }
        50% {
            transform: scale(1.18);
            filter: drop-shadow(0 0 18px var(--glow))
                drop-shadow(0 0 32px var(--glow));
        }
    }

    @keyframes emojiBounce {
        0%,
        100% {
            transform: scale(1.2) translateY(0);
        }
        50% {
            transform: scale(1.25) translateY(-2px);
        }
    }

    .blob-svg {
        opacity: var(--opacity, 0.8);
        transform: translate(var(--ox, 0), var(--oy, 0))
            rotate(var(--rot, 0deg)) scale(var(--scale, 1));
        transition:
            opacity 0.3s ease,
            filter 0.3s ease;
    }

    .blob-path {
        transition: d 0.4s ease;
    }

    .emotion-btn {
        transition:
            transform 0.25s cubic-bezier(0.34, 1.56, 0.64, 1),
            filter 0.25s ease;
    }

    .emotion-btn:hover,
    .emotion-btn.auto-hovered {
        animation: hoverPulse 1.2s ease-in-out infinite;
        z-index: 10;
    }

    .emotion-btn.selected {
        transform: scale(1.12);
        animation: selectedGlow 2s ease-in-out infinite;
    }

    .emotion-btn.selected .blob-svg:first-child {
        filter: drop-shadow(0 0 14px var(--glow))
            drop-shadow(0 0 28px var(--glow));
    }

    .emotion-btn:focus-visible {
        outline: 2px solid var(--glow);
        outline-offset: 3px;
    }

    .emotion-btn:hover .blob-svg,
    .emotion-btn.auto-hovered .blob-svg {
        opacity: calc(var(--opacity, 0.8) * 1.2);
    }

    .emotion-btn:hover .blob-svg:first-child,
    .emotion-btn.auto-hovered .blob-svg:first-child {
        filter: drop-shadow(0 0 16px var(--glow))
            drop-shadow(0 0 32px var(--glow)) drop-shadow(0 0 48px var(--glow));
    }

    .emotion-btn:hover .emoji,
    .emotion-btn.auto-hovered .emoji {
        animation: emojiBounce 0.8s ease-in-out infinite;
    }

    .emotion-btn:hover .name,
    .emotion-btn.auto-hovered .name {
        text-shadow:
            0 1px 3px rgba(0, 0, 0, 0.9),
            0 0 12px var(--primary);
    }
</style>
