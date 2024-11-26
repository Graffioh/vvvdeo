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
		return new Response('Hello');
	},
	async queue(batch) {
		for (const message of batch.messages) {
			try {
				const videoKey = message.body.object.key;

				const response = await fetch('https://6593-2-45-237-19.ngrok-free.app/video-upload-complete', {
					method: 'POST',
					headers: { 'Content-Type': 'application/json' },
					body: JSON.stringify({ videoKey: videoKey, videoStatus: 'uploaded' }),
				});

				if (!response.ok) {
					throw new Error(`Failed to process video: ${response.statusText}`);
				}

				message.ack();
			} catch (err) {
				console.error('Error processing message:', err);

				message.retry();
			}
		}
	},
};
