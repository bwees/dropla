import { finaliseUpload, getInstanceConfig } from '$lib/sdk';
import * as tus from 'tus-js-client';

/**
 * The request-body ceiling is advertised by the server rather than compiled in,
 * because it is a property of the origin the app was loaded from: the public one
 * may sit behind a proxy that caps request bodies while the private one reaches
 * the instance directly. A static build cannot know which it is serving.
 *
 * Cached for the page's lifetime, since it cannot change without a reload, and
 * resolved as uncapped on failure so a blip does not silently halve throughput.
 */
let ceiling: Promise<number> | null = null;

function maxRequestBytes(): Promise<number> {
	ceiling ??= getInstanceConfig()
		.then((config) => config.maxRequestBytes)
		.catch(() => 0);
	return ceiling;
}

export type UploadStatus = 'pending' | 'uploading' | 'done' | 'error';

/**
 * Throughput is sampled on a fixed cadence rather than derived from tus's own
 * progress callbacks, which fire at whatever interval the browser chooses.
 * Sampling independently is what lets a stalled upload read as zero once the
 * window empties, instead of freezing at the last rate it managed.
 */
const SAMPLE_INTERVAL_MS = 500;
const RATE_WINDOW_MS = 5000;

export class UploadItem {
	file: File;
	status = $state<UploadStatus>('pending');
	uploaded = $state(0);
	error = $state('');
	bytesPerSecond = $state(0);
	stalled = $state(false);

	private upload: tus.Upload | null = null;
	private samples: { at: number; bytes: number }[] = [];

	constructor(file: File) {
		this.file = file;
	}

	get progress() {
		if (this.file.size === 0) return this.status === 'done' ? 100 : 0;
		return Math.min(100, Math.round((this.uploaded / this.file.size) * 100));
	}

	get remainingSeconds() {
		if (!this.bytesPerSecond) return Infinity;
		return (this.file.size - this.uploaded) / this.bytesPerSecond;
	}

	sample(now: number) {
		this.samples.push({ at: now, bytes: this.uploaded });

		const cutoff = now - RATE_WINDOW_MS;
		while (this.samples.length > 2 && this.samples[0].at < cutoff) this.samples.shift();

		const first = this.samples[0];
		const last = this.samples[this.samples.length - 1];
		const spanned = last.at - first.at;

		this.bytesPerSecond =
			spanned > 0 ? Math.max(0, (last.bytes - first.bytes) / (spanned / 1000)) : 0;
		// Only call it stalled once a full window has gone by with nothing moving,
		// so the first seconds of an upload are not reported as a stall.
		this.stalled = spanned >= RATE_WINDOW_MS && this.bytesPerSecond === 0 && this.uploaded > 0;
	}

	resetRate() {
		this.samples = [];
		this.bytesPerSecond = 0;
		this.stalled = false;
	}

	attach(upload: tus.Upload) {
		this.upload = upload;
	}

	abort() {
		void this.upload?.abort(true);
	}
}

export type UploadBatchOptions = {
	shareToken: string;
	uploaderLabel?: string;
	onFileDone?: () => void;
};

/**
 * Uploads a batch of files over tus, then finalises the session.
 *
 * The session id groups the batch so the server can report per-file outcomes and
 * send exactly one notification, however many files were dropped.
 */
export class UploadBatch {
	items = $state<UploadItem[]>([]);
	sessionId = crypto.randomUUID();
	finished = $state(false);

	constructor(
		files: File[],
		private options: UploadBatchOptions
	) {
		this.items = files.map((f) => new UploadItem(f));
	}

	get overallProgress() {
		const total = this.items.reduce((sum, i) => sum + i.file.size, 0);
		if (total === 0) return 0;
		const done = this.items.reduce((sum, i) => sum + i.uploaded, 0);
		return Math.min(100, Math.round((done / total) * 100));
	}

	get allSettled() {
		return this.items.every((i) => i.status === 'done' || i.status === 'error');
	}

	get active() {
		return this.items.find((i) => i.status === 'uploading') ?? null;
	}

	// Uploads run one at a time, so the active file's rate is also the batch's.
	get bytesPerSecond() {
		return this.active?.bytesPerSecond ?? 0;
	}

	get remainingSeconds() {
		const rate = this.bytesPerSecond;
		if (!rate) return Infinity;

		const total = this.items.reduce((sum, i) => sum + i.file.size, 0);
		const done = this.items.reduce((sum, i) => sum + i.uploaded, 0);
		return (total - done) / rate;
	}

	async run() {
		const maxBytes = await maxRequestBytes();

		const ticker = setInterval(() => {
			const now = Date.now();
			for (const item of this.items) {
				if (item.status === 'uploading') item.sample(now);
			}
		}, SAMPLE_INTERVAL_MS);

		try {
			// Sequential rather than parallel: several concurrent multi-gigabyte uploads
			// compete for the same bandwidth and make every one of them look stalled.
			for (const item of this.items) {
				await this.uploadOne(item, maxBytes);
				this.options.onFileDone?.();
			}
		} finally {
			clearInterval(ticker);
		}

		try {
			await finaliseUpload(this.options.shareToken, { uploadSessionId: this.sessionId });
		} catch {
			// A failed finalise costs a notification, not the files themselves.
		}
		this.finished = true;
	}

	private uploadOne(item: UploadItem, maxBytes: number): Promise<void> {
		return new Promise((resolve) => {
			item.resetRate();
			item.status = 'uploading';

			const upload = new tus.Upload(item.file, {
				endpoint: '/api/uploads/',
				// An uncapped origin takes the file in a single request. Splitting it
				// costs a round trip per boundary and, on network storage, a forced
				// flush — so it is done only where a proxy would otherwise refuse the
				// body outright.
				chunkSize: maxBytes > 0 ? maxBytes : Number.POSITIVE_INFINITY,
				removeFingerprintOnSuccess: true,
				retryDelays: [0, 1000, 3000, 5000, 10000],
				metadata: {
					filename: item.file.name,
					filetype: item.file.type,
					shareToken: this.options.shareToken,
					uploadSessionId: this.sessionId,
					uploaderLabel: this.options.uploaderLabel ?? ''
				},
				onProgress: (sent) => {
					item.uploaded = sent;
				},
				onSuccess: () => {
					item.uploaded = item.file.size;
					item.status = 'done';
					item.resetRate();
					resolve();
				},
				onError: (error) => {
					item.status = 'error';
					item.error = readableTusError(error);
					item.resetRate();
					// Terminate so the server drops this upload's quota reservation now
					// rather than at the next abandoned-upload sweep.
					void upload.abort(true).catch(() => {});
					resolve();
				}
			});

			item.attach(upload);
			upload.start();
		});
	}
}

// tus surfaces the server's response inside the error message; pull out the
// readable part so a quota refusal does not appear as a wall of protocol detail.
function readableTusError(error: Error | tus.DetailedError): string {
	const detailed = error as tus.DetailedError;
	const body = detailed.originalResponse?.getBody?.();

	if (body) {
		try {
			const parsed = JSON.parse(body);
			if (parsed.detail || parsed.title) return parsed.detail ?? parsed.title;
		} catch {
			const trimmed = body.trim();
			if (trimmed && trimmed.length < 300) return trimmed;
		}
	}

	if (detailed.originalResponse?.getStatus?.() === 413) {
		return 'Refused: this would exceed the available space.';
	}
	return error.message || 'Upload failed.';
}
