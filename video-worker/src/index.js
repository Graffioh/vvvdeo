/**
 * Welcome to Cloudflare Workers! This is your first worker.
 *
 * - Run `npm run dev` in your terminal to start a development server
 * - Open a browser tab at http://localhost:8787/ to see your worker in action
 * - Run `npm run deploy` to publish your worker
 *
 * Learn more at https://developers.cloudflare.com/workers/
 */

export default {
	async fetch() {
		return new Response('vvvdeo video worker');
	},
	async queue(batch) {
		for (const message of batch.messages) {
			try {
				const videoKey = message.body.object.key;

				if (videoKey.includes('videos/')) {
					const videoUploadResponse = await fetch('https://6593-2-45-237-19.ngrok-free.app/video-upload-complete', {
						method: 'POST',
						headers: { 'Content-Type': 'application/json' },
						body: JSON.stringify({ videoKey: videoKey, videoStatus: 'uploaded' }),
					});

					if (!videoUploadResponse.ok) {
						throw new Error(`Failed to process video: ${videoUploadResponse.statusText}`);
					}
				} else if (videoKey.includes('frames/')) {
					const frameExtractionResponse = await fetch('https://6593-2-45-237-19.ngrok-free.app/frames-extraction-complete', {
						method: 'POST',
						headers: { 'Content-Type': 'application/json' },
						body: JSON.stringify({ videoKey: videoKey, videoStatus: 'extracted' }),
					});

					if (!frameExtractionResponse.ok) {
						throw new Error(`Failed to extract frames: ${frameExtractionResponse.statusText}`);
					}
				}

				message.ack();
			} catch (err) {
				console.error('Error processing message:', err);

				message.retry();
			}
		}
	},
};
