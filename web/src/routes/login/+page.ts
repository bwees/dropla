import { getSetupStatus } from '$lib/sdk';
import { requireGuest } from '$lib/utils/auth';
import type { PageLoad } from './$types';

export const load: PageLoad = async () => {
	await requireGuest();

	let setupRequired: boolean;
	try {
		setupRequired = (await getSetupStatus()).setupRequired ?? false;
	} catch {
		setupRequired = false;
	}

	return { setupRequired };
};
