// The public Share page is the only surface strangers see: no auth guard, no app
// chrome, and deliberately no hint that an admin UI exists.
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ params }) => {
	return { token: params.token };
};
