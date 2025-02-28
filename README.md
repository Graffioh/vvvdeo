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

The most fun feature to implement! I learned a lot about building a custom backend segmentation system by leveraging existing models (thanks to Claude for the code).

It was implemented using Cloudflare Workers and Cloudflare Queues for asynchronous processing. However it's currently not working since a lot has changed, and the backend is not deployed cause I'm broke.

Here's a rough diagram of the whole async workflow (shoutout to [moni](https://x.com/fr3fou) for the help):

<img width="714" alt="image" src="https://github.com/user-attachments/assets/d5d7dee8-e98f-4532-8372-fe6f1dd16c8b" />

## Local Setup

### Frontend

Inside the ```/frontend``` folder:

- Create a .env file and add the following line:
  ```VITE_BACKEND_URL=http://localhost:8080```
- Run the development server:
  ```npm run dev```

### Backend

Inside the ```/backend``` folder:

- Create a .env file and add the following line:
  ```APP_ENV=DEV```
- Start the backend server:
  ```go run main.go```

#### Segmentation

Inside the ```/sam2seg``` folder:

- Install SAM2 by following the official instructions: https://github.com/facebookresearch/sam2#installation
- Run this to install required packages:
  ```pip install -r requirements.txt```
- Run the Python segmentation backend:
  ```python sam2segmentation.py```

**NOTE for Mac users**: If you are on M1/M2 the model will use MPS and fallback to CPU for unsupported ops, but sometimes MPS is much slower than CPU so use at your own risk.

----

I hope someone finds this messy source code useful :)
