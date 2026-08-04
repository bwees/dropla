import type { ShareDto } from '$lib/sdk';

/**
 * Extracts the link token a Share's uploads and downloads are addressed by.
 *
 * Guarded rather than assumed: when the API or a proxy in front of it returns
 * something unexpected, reading `.path` off it produced a bare TypeError with no
 * indication of what went wrong. Failing with a readable message means the caller
 * can surface it as a toast.
 */
export function shareToken(share: ShareDto | undefined | null): string {
	const path = share?.path;
	if (!path || !path.startsWith('/s/')) {
		throw new Error('The server returned an unexpected response for this share.');
	}
	return path.slice('/s/'.length);
}
