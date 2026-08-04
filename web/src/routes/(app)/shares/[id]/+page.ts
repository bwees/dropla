import { authenticate } from '$lib/utils/auth';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ params }) => {
	await authenticate();
	return { id: params.id };
};
