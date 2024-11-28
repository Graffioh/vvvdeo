import os
import json
import numpy as np
import torch
from flask import Flask, request, jsonify, send_file
from sam2.build_sam import build_sam2_video_predictor
from PIL import Image, ImageOps, ImageFilter
import time
import cv2
import supervision as sv
import subprocess
import shutil
import requests
import tempfile
import zipfile
import logging
from logging.handlers import RotatingFileHandler

app = Flask(__name__)


# Configure Logging
log_formatter = logging.Formatter(
    '%(asctime)s - %(levelname)s - %(message)s [in %(pathname)s:%(lineno)d]'
)
log_handler = RotatingFileHandler('app.log', maxBytes=1000000, backupCount=3)
log_handler.setFormatter(log_formatter)
log_handler.setLevel(logging.DEBUG)

app.logger.addHandler(log_handler)
app.logger.setLevel(logging.DEBUG)

# Replace all print statements with logging
app.logger.info("Starting the application...")

def clear_directory(directory_path):
    for filename in os.listdir(directory_path):
        file_path = os.path.join(directory_path, filename)
        if os.path.isfile(file_path):
            os.remove(file_path)
        elif os.path.isdir(file_path):
            shutil.rmtree(file_path)

clear_directory("./static/")


device = torch.device("cuda" if torch.cuda.is_available() else "cpu")
sam2_checkpoint = "./src/sam-2/checkpoints/sam2.1_hiera_small.pt"
model_cfg = "./configs/sam2.1/sam2.1_hiera_s.yaml"
predictor = build_sam2_video_predictor(model_cfg, sam2_checkpoint, device=device)

def add_logo_to_frame(frame, logo_img, position='top-right', padding=50):
    if logo_img is None:
        return frame

    frame_h, frame_w = frame.shape[:2]
    logo_h, logo_w = logo_img.shape[:2]

    if position == 'top-right':
        x = frame_w - logo_w - padding
        y = padding
    elif position == 'top-left':
        x = padding
        y = padding
    elif position == 'bottom-right':
        x = frame_w - logo_w - padding
        y = frame_h - logo_h - padding
    elif position == 'bottom-left':
        x = padding
        y = frame_h - logo_h - padding
    else:
        raise ValueError("Unsupported position. Use 'top-right', 'top-left', 'bottom-right', or 'bottom-left'.")

    roi = frame[y:y+logo_h, x:x+logo_w]
    logo_alpha = logo_img[:, :, 3] / 255.0
    for c in range(3):
        roi[:, :, c] = (1 - logo_alpha) * roi[:, :, c] + logo_alpha * logo_img[:, :, c]

    frame[y:y+logo_h, x:x+logo_w] = roi
    return frame

def apply_masked_overlay(frame, masks, overlay_img):
    result_frame = frame.copy()

    if overlay_img is None:
        return result_frame

    for mask in masks:
        if not mask.any():
            continue

        y_indices, x_indices = np.where(mask)
        if len(y_indices) == 0 or len(x_indices) == 0:
            continue

        y_min, y_max = np.min(y_indices), np.max(y_indices)
        x_min, x_max = np.min(x_indices), np.max(x_indices)
        mask_height = y_max - y_min + 1
        mask_width = x_max - x_min + 1

        resized_overlay = cv2.resize(overlay_img, (mask_width, mask_height))

        current_mask = mask[y_min:y_max+1, x_min:x_max+1]

        overlay_color = resized_overlay[:, :, :3]
        overlay_alpha = resized_overlay[:, :, 3] / 255.0
        overlay_alpha = np.expand_dims(overlay_alpha, axis=2)
        current_mask = np.expand_dims(current_mask, axis=2)

        region = result_frame[y_min:y_max+1, x_min:x_max+1]
        blended = region * (1 - (overlay_alpha * current_mask)) + \
                 overlay_color * (overlay_alpha * current_mask)

        result_frame[y_min:y_max+1, x_min:x_max+1] = blended.astype(np.uint8)

    return result_frame

def load_and_prepare_image(image_path, required_format="RGBA"):
    if not os.path.exists(image_path):
        raise ValueError(f"Image file not found: {image_path}")

    image = cv2.imread(image_path, cv2.IMREAD_UNCHANGED)
    if image is None:
        raise ValueError(f"Failed to load image: {image_path}")

    if len(image.shape) == 2:
        image = cv2.cvtColor(image, cv2.COLOR_GRAY2RGB)

    if required_format == "RGBA":
        if len(image.shape) == 3 and image.shape[2] == 3:
            alpha_channel = np.ones(image.shape[:2], dtype=image.dtype) * 255
            image = cv2.merge([image, alpha_channel])
        elif len(image.shape) != 3 or image.shape[2] != 4:
            raise ValueError(f"Image must be in RGB or RGBA format. Current shape: {image.shape}")

    return image

def get_path_from_presignedurl(key) -> str:
        url = f"http://localhost:8080/presigned-get-url?key={key}"
        try:
            response = requests.get(url)
            response.raise_for_status()
            response_data = response.json()

            path = response_data.get('presignedUrl')
            if not path:
                raise RuntimeError("Response did not contain 'presignedUrl'.")
            return path
        except requests.RequestException as e:
            raise RuntimeError(f"Failed to fetch presigned URL: {e}")

input_video = "./static/video_result.mp4"
output_video = "./static/output_compatible.mp4"

def download_video(video_url, temp_dir):
    local_video_path = os.path.join(temp_dir, "downloaded_video.mp4")

    try:
        with requests.get(video_url, stream=True) as response:
            response.raise_for_status()
            with open(local_video_path, 'wb') as f:
                shutil.copyfileobj(response.raw, f)
    except requests.exceptions.RequestException as e:
        raise ValueError(f"Failed to download video: {e}")

    return local_video_path

def download_and_extract_zip(zip_url, temp_dir):
    # Define paths
    zip_path = os.path.join(temp_dir, "downloaded_archive.zip")
    extract_dir = os.path.join(temp_dir, "frames")

    # Download the .zip file
    try:
        with requests.get(zip_url, stream=True) as response:
            response.raise_for_status()  # Raise an error for HTTP status codes 4xx/5xx
            with open(zip_path, 'wb') as f:
                shutil.copyfileobj(response.raw, f)  # Save the streamed content
    except requests.exceptions.RequestException as e:
        raise ValueError(f"Failed to download zip file: {e}")

    # Extract the .zip file
    try:
        with zipfile.ZipFile(zip_path, 'r') as zip_ref:
            zip_ref.extractall(extract_dir)  # Extract into the "frames/" directory
    except zipfile.BadZipFile:
        raise ValueError("The downloaded file is not a valid .zip archive")

    return extract_dir

@app.route("/predict-frames", methods=["POST"])
def predict_frames():
    try:
        app.logger.info("Received request for /predict-frames")
        with tempfile.TemporaryDirectory() as temp_dir:
            # Log initial request data
            app.logger.debug("Temp directory created: %s", temp_dir)

            # get r2 bucket object url
            file_key = request.args.get('key')
            if not file_key:
                app.logger.error("Missing 'file_key' in request body")
                return jsonify({"error": "Missing 'file_key' in request body"}), 400

            app.logger.info(f"File key received: {file_key}")

            cloudflare_video_path = get_path_from_presignedurl("videos/" + file_key)
            app.logger.debug(f"Cloudflare video path: {cloudflare_video_path}")

            video_name = file_key
            local_video_path = download_video(cloudflare_video_path, temp_dir)

            app.logger.debug(f"Local video path: {local_video_path}")

            # Handling potential errors in video info
            try:
                video_info = sv.VideoInfo.from_video_path(local_video_path)
                app.logger.debug("Video info successfully retrieved")
            except AttributeError as e:
                app.logger.exception("Invalid video name format")
                raise ValueError("Invalid video name format") from e
            except FileNotFoundError as e:
                app.logger.exception(f"Video file not found: {video_name}")
                raise ValueError(f"Video file not found: {video_name}") from e

            cloudflare_frames_path = get_path_from_presignedurl("frames/" + file_key + ".zip")
            local_frames_path = download_and_extract_zip(cloudflare_frames_path, temp_dir)
            app.logger.debug(f"Frames extracted to: {local_frames_path}")

            # Rest of your prediction logic, log key steps
            inference_state = predictor.init_state(video_path=local_frames_path)
            predictor.reset_state(inference_state)

            app.logger.info("Inference state initialized and reset")

            data = request.get_json()
            app.logger.debug("Request JSON: %s", data)

            points = data.get("coordinates", [])
            labels = data.get("labels", [])
            overlay_img_name = request.args.get("image")

            # Example: Log when applying overlays
            if overlay_img_name:
                app.logger.info(f"Overlay image: {overlay_img_name}")
            else:
                app.logger.warning("No overlay image provided")

            # Additional logging in your processing loop
            for frame_idx, object_ids, mask_logits in predictor.propagate_in_video(inference_state):
                app.logger.debug(f"Processing frame index: {frame_idx}")

                if not os.path.exists(frames_paths[frame_idx]):
                    app.logger.warning(f"Frame '{frames_paths[frame_idx]}' does not exist.")
                    continue

                # Further processing...
            app.logger.info("Frames processed successfully")

        # Handle video encoding and logging
        try:
            subprocess.run(command, check=True)
            app.logger.info(f"Re-encoded video saved as: {output_video}")
        except subprocess.CalledProcessError as e:
            app.logger.exception("Error during re-encoding")

        return send_file(output_video, mimetype='video/mp4', as_attachment=True, download_name='processed_video.mp4')
    except Exception as e:
        app.logger.exception("Error occurred in /predict-frames")
        return jsonify({"error": str(e), "status": "error"}), 500

if __name__ == "__main__":
    app.run(host="0.0.0.0", port=9000)
