# vvvdeo

Trim or speed up videos with a click by downloading them from social media thanks to cobalt or choosing a local video.

This project was mostly for fun/experimentation, so I'm open-sourcing it, just in case you're curious about the code or want to take inspiration for your own projects.

# Frontend

The frontend (plain HTML/CSS & JS) is deployed on Cloudflare Pages using Vite for the build process.

# Backend

The backend (written in Go) is dockerized and deployed on Fly.io.

## Trim and Speedup Features

- **Trimming** is implemented with `ffmpeg.wasm`, so everything is done client-side. I wanted to try `ffmpeg.wasm`, and overall, it's really cool—but it still has room for improvement.
- **Speedup** is done server-side with `ffmpeg`.

## SAM2 Segmentation

The most fun feature to implement! I learned a lot about building a custom backend segmentation system by leveraging existing models.

It was implemented using Cloudflare Workers and Cloudflare Queues for asynchronous processing. However it's currently not working since a lot has changed, and the backend is not deployed cause I'm broke.

Here's a rough diagram of the whole async workflow (shoutout to [moni](https://x.com/fr3fou) for the help):

<img width="714" alt="image" src="https://github.com/user-attachments/assets/d5d7dee8-e98f-4532-8372-fe6f1dd16c8b" />

I hope someone finds this messy source code useful :)
