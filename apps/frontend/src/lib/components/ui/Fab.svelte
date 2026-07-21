<script lang="ts">
import { Plus } from 'lucide-svelte';

interface Props {
	/** Click handler */
	onclick?: () => void;
	/** Accessible label */
	label?: string;
	/** Extra classes */
	class?: string;
}

let {
	onclick,
	label = 'Add',
	class: className = '',
}: Props = $props();
</script>

<button
	class="fab btn btn-primary btn-circle btn-lg shadow-xl hover:scale-105 active:scale-95 transition-all duration-200 {className}"
	{onclick}
	aria-label={label}
	title={label}
>
	<Plus class="w-6 h-6" strokeWidth={2.5} />
</button>
