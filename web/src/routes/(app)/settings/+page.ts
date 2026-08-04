import { authenticate } from '$lib/utils/auth';
import type { PageLoad } from './$types';

export const load: PageLoad = async () => {
	const user = await authenticate();
	return { user };
};
