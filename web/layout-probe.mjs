// Measures real layout in a browser instead of reasoning about CSS.
// Reports horizontal page overflow and any element wider than its container.
import { chromium } from 'playwright';

const BASE = 'http://localhost:5173';
const [, , shareId] = process.argv;

const findOverflow = () => ({
	page: {
		scrollWidth: document.documentElement.scrollWidth,
		clientWidth: document.documentElement.clientWidth
	},
	offenders: Array.from(document.querySelectorAll('body *'))
		.map((el) => {
			const parent = el.parentElement;
			if (!parent) return null;
			const r = el.getBoundingClientRect();
			const p = parent.getBoundingClientRect();
			const overflowRight = Math.round(r.right - p.right);
			if (overflowRight <= 1 || r.width === 0) return null;
			const style = getComputedStyle(parent);
			// A parent that scrolls or clips deliberately is not a bug.
			if (
				style.overflowX === 'auto' ||
				style.overflowX === 'scroll' ||
				style.overflowX === 'hidden'
			)
				return null;
			// display:contents parents have no box, so the comparison is meaningless.
			if (style.display === 'contents' || p.width === 0) return null;
			return {
				tag: el.tagName.toLowerCase(),
				cls: (el.className || '').toString().slice(0, 70),
				width: Math.round(r.width),
				parentWidth: Math.round(p.width),
				overflowRight,
				text: (el.textContent || '').trim().slice(0, 40)
			};
		})
		.filter(Boolean)
		.sort((a, b) => b.overflowRight - a.overflowRight)
		.slice(0, 12)
});

const browser = await chromium.launch();
let bad = 0;

for (const viewport of [
	{ width: 1440, height: 900 },
	{ width: 1100, height: 900 },
	{ width: 900, height: 900 },
	{ width: 420, height: 900 }
]) {
	const ctx = await browser.newContext({ viewport });
	const page = await ctx.newPage();

	await page.goto(`${BASE}/login`, { waitUntil: 'networkidle' });
	await page.fill('#username', 'bwees');
	await page.fill('#password', 'correcthorse');
	await page.click('button[type=submit]');
	await page.waitForURL(`${BASE}/`, { timeout: 15000 });

	for (const path of ['/', `/shares/${shareId}`, '/settings']) {
		await page.goto(`${BASE}${path}`, { waitUntil: 'networkidle' });
		await page.waitForTimeout(400);
		const r = await page.evaluate(findOverflow);

		const scrolls = r.page.scrollWidth > r.page.clientWidth + 1;
		const label = `${viewport.width}px ${path}`;
		if (scrolls || r.offenders.length) {
			bad++;
			console.log(`\nFAIL ${label}`);
			if (scrolls)
				console.log(`  page scrolls sideways: ${r.page.scrollWidth} > ${r.page.clientWidth}`);
			for (const o of r.offenders)
				console.log(
					`  +${o.overflowRight}px  <${o.tag}> w=${o.width} parent=${o.parentWidth}  "${o.text}"\n            ${o.cls}`
				);
		} else {
			console.log(`PASS ${label}`);
		}
	}
	await ctx.close();
}

await browser.close();
console.log(bad === 0 ? '\nNO OVERFLOW AT ANY WIDTH' : `\n${bad} VIEW(S) OVERFLOWING`);
process.exit(bad === 0 ? 0 : 1);
