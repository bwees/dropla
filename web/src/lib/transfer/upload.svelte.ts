import { finaliseUpload } from '$lib/sdk';
import * as tus from 'tus-js-client';

export type UploadStatus = 'pending' | 'uploading' | 'done' | 'error';

export class UploadItem {
	file: File;
	status = $state<UploadStatus>('pending');
	uploaded = $state(0);
	error = $state('');

	private upload: tus.Upload | null = null;

	constructor(file: File) {
		this.file = file;
	}

	get progress() {
		if (this.file.size === 0) return this.status === 'done' ? 100 : 0;
		return Math.min(100, Math.round((this.uploaded / this.file.size) * 100));
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

	async run() {
		// Sequential rather than parallel: several concurrent multi-gigabyte uploads
		// compete for the same bandwidth and make every one of them look stalled.
		for (const item of this.items) {
			await this.uploadOne(item);
			this.options.onFileDone?.();
		}

		try {
			await finaliseUpload(this.options.shareToken, { uploadSessionId: this.sessionId });
		} catch {
			// A failed finalise costs a notification, not the files themselves.
		}
		this.finished = true;
	}

	private uploadOne(item: UploadItem): Promise<void> {
		return new Promise((resolve) => {
			item.status = 'uploading';

			const upload = new tus.Upload(item.file, {
				endpoint: '/api/uploads/',
				// tus-js-client defaults chunkSize to Infinity, which sends one huge
				// PATCH that proxy body limits reject. 32 MB stays under Cloudflare's
				// 100 MB per-request cap.
				chunkSize: 32 * 1024 * 1024,
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
					resolve();
				},
				onError: (error) => {
					item.status = 'error';
					item.error = readableTusError(error);
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
