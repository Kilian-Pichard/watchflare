import { Home, Server } from 'lucide-svelte';

export const navItems = [
	{ href: '/', label: 'Dashboard', icon: Home },
	{ href: '/hosts', label: 'Hosts', icon: Server },
];

export const settingsItems = [
	{ href: '/settings', label: 'General' },
	{ href: '/settings/notifications', label: 'Notifications' },
];
