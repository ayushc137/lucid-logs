<!--
  @deprecated UNUSED COMPONENT - Moved to graveyard on 2026-01-22

  This component provided view switching between Timeline and Agenda views.
  The agenda view was removed to simplify the UI, making this switcher unnecessary.
-->
<script lang="ts">
import { AlignHorizontalJustifyStart, List } from 'lucide-svelte';

export type TimelineView = 'timeline' | 'agenda';

interface Props {
	currentView: TimelineView;
	onViewChange: (view: TimelineView) => void;
}

let { currentView, onViewChange }: Props = $props();
</script>

<div class="join border border-base-300/60 rounded-lg">
    <button
        type="button"
        class="join-item btn btn-xs btn-square {currentView === 'timeline'
            ? 'btn-primary'
            : 'btn-ghost'}"
        onclick={() => onViewChange("timeline")}
        title="Timeline View"
    >
        <AlignHorizontalJustifyStart class="w-3.5 h-3.5" />
    </button>
    <button
        type="button"
        class="join-item btn btn-xs btn-square {currentView === 'agenda'
            ? 'btn-primary'
            : 'btn-ghost'}"
        onclick={() => onViewChange("agenda")}
        title="Agenda View"
    >
        <List class="w-3.5 h-3.5" />
    </button>
</div>
