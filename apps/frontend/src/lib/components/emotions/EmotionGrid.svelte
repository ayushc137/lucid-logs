<script lang="ts">
    /**
     * EMOTION GRID
     * - OpenMoji font
     * - Tooltip portaled to body (outside modal) via action
     * - panzoom for smooth pan/zoom
     * - Tailwind CSS styling
     */
    import { onMount, onDestroy } from "svelte";
    import panzoom, { type PanZoom } from "panzoom";
    import * as blobs2 from "blobs/v2";
    import {
        type Emotion,
        QUADRANT_COLORS,
        EMOTIONS,
        seededRandom,
    } from "./emotionData";
    import { ZoomIn, ZoomOut, RotateCcw } from "lucide-svelte";

    interface Props {
        selectedEmotion?: Emotion | null;
        onSelect?: (emotion: Emotion) => void;
        showIndicators?: boolean;
        onHoveredChange?: (emotion: Emotion | null) => void;
    }

    let {
        selectedEmotion = null,
        onSelect,
        showIndicators = false,
        onHoveredChange,
    }: Props = $props();

    // Hover state
    let hoveredEmotion = $state<Emotion | null>(null);
    let tooltipPosition = $state<{ x: number; y: number } | null>(null);
    let isMouseHovering = $state(false);

    // Panzoom
    let viewportElement: HTMLDivElement;
    let gridElement: HTMLDivElement;
    let pzInstance: PanZoom | null = null;
    let zoomLevel = $state(1);

    // Center detection for auto-hover
    let centerEmotion = $state<Emotion | null>(null);

    function findCenterEmotion() {
        if (!viewportElement || !gridElement) return;

        const viewportRect = viewportElement.getBoundingClientRect();
        const centerX = viewportRect.left + viewportRect.width / 2;
        const centerY = viewportRect.top + viewportRect.height / 2;

        const buttons = gridElement.querySelectorAll(".emotion-btn");
        let closest: { emotion: Emotion; distance: number } | null = null;

        buttons.forEach((btn) => {
            const rect = btn.getBoundingClientRect();
            const btnCenterX = rect.left + rect.width / 2;
            const btnCenterY = rect.top + rect.height / 2;
            const distance = Math.sqrt(
                Math.pow(centerX - btnCenterX, 2) +
                    Math.pow(centerY - btnCenterY, 2),
            );

            const emotionId = btn.getAttribute("data-emotion-id");
            const emotion = EMOTIONS.find((e) => e.id === emotionId);

            if (emotion && (!closest || distance < closest.distance)) {
                closest = { emotion, distance };
            }
        });

        if (closest && closest.distance < 100) {
            centerEmotion = closest.emotion;
            if (!isMouseHovering) {
                hoveredEmotion = closest.emotion;
                onHoveredChange?.(closest.emotion);
            }
        }
    }

    onMount(() => {
        if (gridElement) {
            pzInstance = panzoom(gridElement, {
                maxZoom: 2.5,
                minZoom: 0.6,
                bounds: true,
                boundsPadding: 0.2,
                smoothScroll: false,
            });

            pzInstance.on("zoom", () => {
                zoomLevel = pzInstance?.getTransform().scale ?? 1;
                findCenterEmotion();
            });

            pzInstance.on("pan", () => {
                findCenterEmotion();
            });

            setTimeout(findCenterEmotion, 100);
        }
    });

    onDestroy(() => {
        pzInstance?.dispose();
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
        setTimeout(findCenterEmotion, 100);
    }

    // Grid layout
    const grid: (Emotion | null)[][] = Array(10)
        .fill(null)
        .map(() => Array(10).fill(null));
    for (const emotion of EMOTIONS) {
        const col = emotion.x < 0 ? emotion.x + 5 : emotion.x + 4;
        const row = emotion.y > 0 ? 5 - emotion.y : Math.abs(emotion.y) + 4;
        if (row >= 0 && row < 10 && col >= 0 && col < 10)
            grid[row][col] = emotion;
    }

    function getEmotionColors(emotion: Emotion) {
        const baseColors = QUADRANT_COLORS[emotion.quadrant];
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

    const emotionLayers = new Map<string, BlobLayer[]>();
    for (const e of EMOTIONS) {
        emotionLayers.set(e.id, generateLayers(e));
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
        isMouseHovering = false;
        tooltipPosition = null;
        if (centerEmotion) {
            hoveredEmotion = centerEmotion;
            onHoveredChange?.(centerEmotion);
        }
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
    <div class="flex justify-between items-center shrink-0">
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

    <!-- Panzoom viewport -->
    <div
        class="flex-1 overflow-hidden rounded-2xl bg-gray-500/[0.02] cursor-grab active:cursor-grabbing"
        bind:this={viewportElement}
    >
        <div
            class="grid grid-cols-10 gap-3.5 p-5 md:gap-4 md:p-6 lg:gap-[18px] lg:p-7"
            bind:this={gridElement}
        >
            {#each grid as row}
                {#each row as emotion}
                    <div class="aspect-square">
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
                                onmouseenter={(e) => onMouseEnter(e, emotion)}
                                onmousemove={onMouseMove}
                                onmouseleave={onMouseLeave}
                                style="--glow: {colors.glow}; --speed: {animSpeed}s;"
                            >
                                <div
                                    class="absolute -inset-[12%] w-[124%] h-[124%] pointer-events-none"
                                >
                                    {#each layers as layer, li}
                                        {@const baseOpacity =
                                            0.45 +
                                            ((layerCount - li) / layerCount) *
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
                                        >{emotion.emoji}</span
                                    >
                                    <span
                                        class="name text-[clamp(0.32rem,0.55vw,0.42rem)] font-semibold text-white uppercase max-w-full overflow-hidden text-ellipsis whitespace-nowrap hidden sm:inline"
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
</div>

<!-- Tooltip - Portaled to body via {#if} to avoid modal clipping -->
{#if hoveredEmotion && tooltipPosition && isMouseHovering && typeof document !== "undefined"}
    {@const colors = QUADRANT_COLORS[hoveredEmotion.quadrant]}
    {@const dots = getIndicatorDots(hoveredEmotion)}
    <div
        class="fixed z-[100000] min-w-[180px] max-w-[260px] bg-white dark:bg-slate-800 rounded-xl shadow-[0_8px_32px_rgba(0,0,0,0.2)] overflow-hidden pointer-events-none animate-[tooltipIn_0.12s_ease-out]"
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
            <p class="m-0 text-xs opacity-70 leading-relaxed">
                {hoveredEmotion.description}
            </p>
            {#if showIndicators && dots.length > 0}
                <div class="flex gap-1 mt-2 flex-wrap">
                    {#each dots as d}
                        <span
                            class="text-[0.58rem] px-1.5 py-0.5 rounded-lg font-medium"
                            style="background: color-mix(in srgb, {d.color} 15%, transparent); color: color-mix(in srgb, {d.color} 80%, #333);"
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

    .blob-svg {
        opacity: var(--opacity, 0.8);
        transform: translate(var(--ox, 0), var(--oy, 0))
            rotate(var(--rot, 0deg)) scale(var(--scale, 1));
    }

    .blob-path {
        transition: d 0.4s ease;
    }

    .emotion-btn:hover,
    .emotion-btn.auto-hovered {
        transform: scale(1.06);
        filter: drop-shadow(0 0 8px var(--glow));
    }

    .emotion-btn.selected {
        transform: scale(1.1);
    }

    .emotion-btn:focus-visible {
        outline: 2px solid var(--glow);
        outline-offset: 3px;
    }

    .emotion-btn:hover .blob-svg:first-child,
    .emotion-btn.auto-hovered .blob-svg:first-child {
        filter: drop-shadow(0 0 10px var(--glow))
            drop-shadow(0 0 20px var(--glow));
    }

    .emotion-btn:hover .emoji,
    .emotion-btn.auto-hovered .emoji {
        transform: scale(1.1);
    }
</style>
