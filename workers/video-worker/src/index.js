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
	async queue(batch, env) {
		for (const message of batch.messages) {
			const key = message.body.object.key;
			try {
				if (key.includes('videos/')) {
					const videoUploadResponse = await fetch(env.BACKEND_URL + '/video-upload-complete', {
						method: 'POST',
						headers: { 'Content-Type': 'application/json' },
						body: JSON.stringify({ videoKey: key, status: 'uploaded' }),
					});

					if (!videoUploadResponse.ok) {
						throw new Error(`Failed to send Video upload notification: ${videoUploadResponse.statusText}`);
					}
				} else if (key.includes('frames/')) {
					const frameExtractionResponse = await fetch(env.BACKEND_URL + '/frames-extraction-complete', {
						method: 'POST',
						headers: { 'Content-Type': 'application/json' },
						body: JSON.stringify({ videoKey: key, status: 'extracted' }),
					});

					if (!frameExtractionResponse.ok) {
						throw new Error(`Failed to send Frames extraction notification: ${frameExtractionResponse.statusText}`);
					}
				}

				message.ack();
			} catch (err) {
				console.error('Error processing message from queue:', err);
				console.error('BACKEND URL: ', env.BACKEND_URL);
				message.retry();
			}
		}
	},
};
