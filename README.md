# vvvdeo

Easily Trim or Speed up videos with a click, and download videos from social media with [cobalt](https://cobalt.tools).

This project was mostly for fun/experimentation, so I'm open-sourcing it, just in case you're curious about the code or want to take inspiration for your own projects.

# Frontend

The frontend (plain HTML/CSS & JS) is deployed on Cloudflare Pages using Vite for the build process.

# Backend

The backend (written in Go) is dockerized and deployed on Fly.io.

## Trim and Speedup Features

- **Trimming** is implemented with `ffmpeg.wasm`, allowing the browser to process the video client-side. I wanted to try `ffmpeg.wasm`, and overall, it's really cool—but it still has room for improvement.
- **Speedup** is done server-side with `ffmpeg`.

## SAM2 Segmentation

**It works only locally**, if you want to run on your computer, follow the instructions in the next section.

This was the most fun feature to implement! I learned a lot about building a custom backend segmentation system by leveraging existing models (thanks to Claude for the code).

### Instructions to run locally

Serving the segmentation backend local API with `FastAPI`.

It uses **Docker**, so install it if you don’t have it already.

- Run docker using whatever you like (I suggest [OrbStack](https://orbstack.dev/) instead of Docker desktop)
- In the root dir run `docker-compose up --build`
- *(Optional but Recommended if you have a NVIDIA GPU)* Install the [NVIDIA Container Toolkit](https://docs.nvidia.com/datacenter/cloud-native/container-toolkit/latest/install-guide.html) and start the stack with `docker compose -f docker-compose.yml -f docker-compose.gpu.yml up --build` to expose CUDA inside the `sam2seg` container
- Open the frontend via `https://localhost:5173` 
- Enjoy

### OLD Asynchronous Workflow

It was implemented using Cloudflare Workers and Cloudflare Queues for asynchronous processing. However currently it's not working (and discontinues) since a lot has changed.

Here's a rough diagram of the whole (old) async workflow (shoutout to [moni](https://x.com/fr3fou) for the help):

<img width="714" alt="image" src="https://github.com/user-attachments/assets/d5d7dee8-e98f-4532-8372-fe6f1dd16c8b" />

This was discontinued because:
- It was a setup built to learn how async processing/workflow works
- Renting a good GPU and keep it running + CloudFlare subscription = too expensive right now

### Demos

[demo1](https://x.com/graffioh/status/1864004204143984955/video/1)
[demo2](https://x.com/graffioh/status/1863957107533320533)

----

I hope someone finds this messy source code useful :)
