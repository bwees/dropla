import { chromium } from 'playwright';

const BASE = 'http://localhost:5173';
const [, , shareId] = process.argv;

const browser = await chromium.launch();

for (const mode of ['light', 'dark']) {
	const ctx = await browser.newContext({
		viewport: { width: 1280, height: 1000 },
		deviceScaleFactor: 1
	});
	const page = await ctx.newPage();
	await page.addInitScript(`localStorage.setItem('dropla-theme', '${mode}')`);

	await page.goto(`${BASE}/login`, { waitUntil: 'networkidle' });
	await page.fill('#username', 'bwees');
	await page.fill('#password', 'correcthorse');
	await page.click('button[type=submit]');
	await page.waitForURL(`${BASE}/`, { timeout: 15000 });

	await page.waitForTimeout(400);
	await page.screenshot({ path: `/tmp/shot-${mode}-dashboard.png`, fullPage: true });

	await page.getByRole('tab', { name: /Upload Boxes/ }).click();
	await page.waitForTimeout(400);
	await page.screenshot({ path: `/tmp/shot-${mode}-boxes.png`, fullPage: true });

	if (shareId) {
		await page.goto(`${BASE}/shares/${shareId}`, { waitUntil: 'networkidle' });
		await page.waitForTimeout(400);
		await page.screenshot({ path: `/tmp/shot-${mode}-share.png`, fullPage: true });
	}

	await page.goto(`${BASE}/settings`, { waitUntil: 'networkidle' });
	await page.waitForTimeout(400);
	await page.screenshot({ path: `/tmp/shot-${mode}-settings.png`, fullPage: true });

	// Open the per-user menu and record its measured size, since a dropdown that
	// renders at the trigger's width would be invisible in a screenshot alone.
	const cog = page.getByRole('button', { name: /^Manage / }).first();
	if (await cog.count()) {
		await cog.click();
		await page.waitForTimeout(350);
		const box = await page.locator('[data-slot=dropdown-menu-content]').first().boundingBox();
		console.log(
			`${mode} user menu: ${box ? `${Math.round(box.width)}x${Math.round(box.height)}` : 'NOT RENDERED'}`
		);
		await page.screenshot({ path: `/tmp/shot-${mode}-usermenu.png` });
	}

	await ctx.close();
}

await browser.close();
console.log('shots written to /tmp/shot-*.png');
