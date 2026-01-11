<script lang="ts">
import { Card } from '$lib/components/ui';
import { Calendar, Clock } from 'lucide-svelte';

interface Props {
	startDate: string;
	endDate: string;
	startTime: string;
	endTime: string;
	isEditing: boolean;
	liveEndTime: boolean;
	useLastTaskStart: boolean;
	lastTaskEndTime: Date | null;
	onStartDateChange: (value: string) => void;
	onEndDateChange: (value: string) => void;
	onStartTimeChange: (value: string) => void;
	onEndTimeChange: (value: string) => void;
	onLiveEndTimeChange: (value: boolean) => void;
	onUseLastTaskStartChange: (value: boolean) => void;
}

let {
	startDate = $bindable(''),
	endDate = $bindable(''),
	startTime = $bindable('09:00:00'),
	endTime = $bindable('10:00:00'),
	isEditing = false,
	liveEndTime = $bindable(false),
	useLastTaskStart = $bindable(false),
	lastTaskEndTime = null,
	onStartDateChange,
	onEndDateChange,
	onStartTimeChange,
	onEndTimeChange,
	onLiveEndTimeChange,
	onUseLastTaskStartChange,
}: Props = $props();
</script>

<Card variant="bordered" class="transition-all duration-200 hover:shadow-md">
    <div class="flex items-center gap-2 mb-3">
        <Calendar class="w-4 h-4 opacity-50" />
        <span class="text-xs font-semibold uppercase opacity-50">Schedule</span>
    </div>

    <div class="bg-base-100 rounded-lg border border-base-300 p-3 space-y-3">
        <!-- Quick toggles (only in create mode) -->
        {#if !isEditing}
            <div class="flex flex-wrap gap-2 pb-2 border-b border-base-200">
                {#if lastTaskEndTime}
                    <label class="flex items-center gap-2 cursor-pointer">
                        <input
                            type="checkbox"
                            class="toggle toggle-xs toggle-primary"
                            bind:checked={useLastTaskStart}
                            onchange={(e) =>
                                onUseLastTaskStartChange(
                                    e.currentTarget.checked,
                                )}
                        />
                        <span
                            class="text-xs {useLastTaskStart
                                ? 'text-primary font-medium'
                                : 'opacity-60'}"
                        >
                            From last task
                        </span>
                    </label>
                {/if}
                <label class="flex items-center gap-2 cursor-pointer ml-auto">
                    <input
                        type="checkbox"
                        class="toggle toggle-xs toggle-success"
                        bind:checked={liveEndTime}
                        onchange={(e) =>
                            onLiveEndTimeChange(e.currentTarget.checked)}
                    />
                    <span
                        class="text-xs flex items-center gap-1 {liveEndTime
                            ? 'text-success font-medium'
                            : 'opacity-60'}"
                    >
                        {#if liveEndTime}
                            <span
                                class="w-1.5 h-1.5 rounded-full bg-success animate-pulse"
                            ></span>
                        {/if}
                        End at Now
                    </span>
                </label>
            </div>
        {/if}

        <div class="space-y-3">
            <div class="grid grid-cols-2 gap-3">
                <div class="form-control">
                    <label class="label py-0 pb-1" for="start-date">
                        <span class="label-text text-xs opacity-50"
                            >Start Date</span
                        >
                    </label>
                    <input
                        id="start-date"
                        type="date"
                        bind:value={startDate}
                        onchange={(e) =>
                            onStartDateChange(e.currentTarget.value)}
                        class="input input-sm input-bordered w-full transition-all duration-200 focus:border-primary"
                        disabled={useLastTaskStart}
                    />
                </div>
                <div class="form-control">
                    <label class="label py-0 pb-1" for="end-date">
                        <span class="label-text text-xs opacity-50"
                            >End Date</span
                        >
                    </label>
                    <input
                        id="end-date"
                        type="date"
                        bind:value={endDate}
                        onchange={(e) => onEndDateChange(e.currentTarget.value)}
                        class="input input-sm input-bordered w-full transition-all duration-200 focus:border-primary {liveEndTime
                            ? 'input-success'
                            : ''}"
                        disabled={liveEndTime}
                    />
                </div>
            </div>
            <div class="grid grid-cols-2 gap-3">
                <div class="form-control">
                    <label class="label py-0 pb-1" for="start-time">
                        <span
                            class="label-text text-xs opacity-50 flex items-center gap-1"
                        >
                            <Clock class="w-3 h-3" /> From
                        </span>
                    </label>
                    <input
                        id="start-time"
                        type="time"
                        step="1"
                        bind:value={startTime}
                        onchange={(e) =>
                            onStartTimeChange(e.currentTarget.value)}
                        class="input input-sm input-bordered w-full transition-all duration-200 focus:border-primary"
                        disabled={useLastTaskStart}
                    />
                </div>
                <div class="form-control">
                    <label class="label py-0 pb-1" for="end-time">
                        <span
                            class="label-text text-xs opacity-50 flex items-center gap-1"
                        >
                            <Clock class="w-3 h-3" /> To
                            {#if liveEndTime}
                                <span class="badge badge-xs badge-success"
                                    >LIVE</span
                                >
                            {/if}
                        </span>
                    </label>
                    <input
                        id="end-time"
                        type="time"
                        step="1"
                        bind:value={endTime}
                        onchange={(e) => onEndTimeChange(e.currentTarget.value)}
                        class="input input-sm input-bordered w-full transition-all duration-200 focus:border-primary {liveEndTime
                            ? 'input-success font-mono'
                            : ''}"
                        disabled={liveEndTime}
                    />
                </div>
            </div>
        </div>
    </div>
</Card>
