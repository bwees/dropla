const UNITS = ['B', 'KB', 'MB', 'GB', 'TB', 'PB'];

export function formatBytes(bytes: number, precision = 1): string {
	if (bytes < 1000) return `${bytes} B`;

	let value = bytes;
	let unit = 0;
	while (value >= 1000 && unit < UNITS.length - 1) {
		value /= 1000;
		unit++;
	}
	return `${value.toFixed(value < 10 ? precision : 0)} ${UNITS[unit]}`;
}

const MULTIPLIERS: Record<string, number> = {
	'': 1,
	b: 1,
	k: 1e3,
	kb: 1e3,
	m: 1e6,
	mb: 1e6,
	g: 1e9,
	gb: 1e9,
	t: 1e12,
	tb: 1e12
};

// Accepts "50GB", "500 mb", "1024", "0". Returns null when unparseable, which
// callers must distinguish from 0 — 0 means unlimited.
export function parseBytes(input: string): number | null {
	const match = /^\s*([\d.]+)\s*([a-zA-Z]*)\s*$/.exec(input);
	if (!match) return null;

	const amount = Number(match[1]);
	if (!Number.isFinite(amount) || amount < 0) return null;

	const multiplier = MULTIPLIERS[match[2].toLowerCase()];
	if (multiplier === undefined) return null;

	return Math.round(amount * multiplier);
}

export function formatDate(iso: string | undefined | null): string {
	if (!iso) return '—';
	return new Date(iso).toLocaleString(undefined, {
		dateStyle: 'medium',
		timeStyle: 'short'
	});
}

// Relative wording for expiry, which is what an Owner actually wants to know.
export function formatExpiry(iso: string | undefined | null): string {
	if (!iso) return 'never';

	const ms = new Date(iso).getTime() - Date.now();
	if (ms <= 0) return 'expired';

	const minutes = Math.round(ms / 60_000);
	if (minutes < 60) return `in ${minutes} min`;

	const hours = Math.round(minutes / 60);
	if (hours < 48) return `in ${hours} h`;

	return `in ${Math.round(hours / 24)} days`;
}
