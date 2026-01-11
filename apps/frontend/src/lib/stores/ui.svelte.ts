/**
 * UI store using Svelte 5 runes
 * Manages global UI state like sidebar, modals, etc.
 */
class UIStore {
	sidebarOpen = $state(true);
	commandPaletteOpen = $state(false);
	activeModal = $state<string | null>(null);

	// Quick log panel
	quickLogOpen = $state(false);
	selectedTemplateId = $state<string | null>(null);

	toggleSidebar() {
		this.sidebarOpen = !this.sidebarOpen;
	}

	toggleCommandPalette() {
		this.commandPaletteOpen = !this.commandPaletteOpen;
	}

	openModal(modalId: string) {
		this.activeModal = modalId;
	}

	closeModal() {
		this.activeModal = null;
	}

	openQuickLog(templateId?: string) {
		this.selectedTemplateId = templateId ?? null;
		this.quickLogOpen = true;
	}

	closeQuickLog() {
		this.quickLogOpen = false;
		this.selectedTemplateId = null;
	}
}

export const uiStore = new UIStore();
