import { Home, Server, Package } from 'lucide-svelte';

export const navItems = [
	{ href: '/', label: 'Dashboard', icon: Home },
	{ href: '/hosts', label: 'Hosts', icon: Server },
	{ href: '/packages', label: 'Packages', icon: Package },
];

export const settingsItems = [
	{ href: '/settings', label: 'General' },
	{ href: '/settings/notifications', label: 'Notifications' },
];
