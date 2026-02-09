import { writable } from 'svelte/store';

// Toast types: 'info', 'success', 'warning', 'error'
function createToastStore() {
	const { subscribe, update } = writable([]);

	let nextId = 0;

	return {
		subscribe,
		add: (message, type = 'info', duration = 5000) => {
			const id = nextId++;
			const toast = { id, message, type };

			update(toasts => [...toasts, toast]);

			// Auto-remove after duration
			if (duration > 0) {
				setTimeout(() => {
					update(toasts => toasts.filter(t => t.id !== id));
				}, duration);
			}

			return id;
		},
		remove: (id) => {
			update(toasts => toasts.filter(t => t.id !== id));
		},
		clear: () => {
			update(() => []);
		}
	};
}

export const toasts = createToastStore();
