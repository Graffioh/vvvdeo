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

def clear_directory(directory_path):
    for filename in os.listdir(directory_path):
        file_path = os.path.join(directory_path, filename)
        if os.path.isfile(file_path):
            os.remove(file_path)
        elif os.path.isdir(file_path):
            shutil.rmtree(file_path)

clear_directory("./static/")

app = Flask(__name__)

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
        with tempfile.TemporaryDirectory() as temp_dir:
            # get r2 bucket object url
            file_key = request.args.get('key')

            if not file_key:
                return jsonify({"error": "Missing 'file_key' in request body"}), 400
            cloudflare_video_path = get_path_from_presignedurl("videos/" + file_key)
            video_name = file_key

            local_video_path = download_video(cloudflare_video_path, temp_dir)
            try:
                video_info = sv.VideoInfo.from_video_path(local_video_path)
                print("VIDEO INFO: ", video_info)
            except AttributeError:
                raise ValueError("Invalid video name format")
            except FileNotFoundError:
                raise ValueError(f"Video file not found: {video_name}")

            cloudflare_frames_path = get_path_from_presignedurl("frames/" + file_key + ".zip")
            local_frames_path = download_and_extract_zip(cloudflare_frames_path, temp_dir)

            inference_state = predictor.init_state(video_path=local_frames_path)
            predictor.reset_state(inference_state)


        print("INFERENCE STATE: " + inference_state)

        data = request.get_json()
        points = data.get("coordinates", [])
        labels = data.get("labels", [])
        overlay_img_name = request.args.get("image")

        mask_opacity = 0
        if not overlay_img_name:
            mask_opacity = 100

        if not points:
            return jsonify({"error": "Missing coordinates"}), 400
        try:
            points = np.array([[p['x'], p['y']] for p in points], dtype=np.float32)
            labels = np.array([l for l in labels], dtype=np.int32)
        except Exception as e:
            return jsonify({"error": f"Error processing points: {str(e)}"}), 500

        ann_frame_idx = 0
        ann_obj_id = 1
        _, out_obj_ids, out_mask_logits = predictor.add_new_points_or_box(
            inference_state=inference_state,
            frame_idx=ann_frame_idx,
            obj_id=ann_obj_id,
            points=points,
            labels=labels,
        )

        colors = ['#FF1493', '#00BFFF', '#FF6347', '#FFD700']
        mask_annotator = sv.MaskAnnotator(
            color=sv.ColorPalette.from_hex(colors),
            color_lookup=sv.ColorLookup.TRACK,
            opacity=mask_opacity
        )

        frames_paths = sorted(sv.list_files_with_extensions(
            directory=local_frames_path,
            extensions=["jpg"]))

        overlay_img = None
        if overlay_img_name is not None:
            overlay_img = load_and_prepare_image(f"./img/{overlay_img_name}")

        logo_img = None
        logo_img_name = "vvvdeo-logo.png"
        if logo_img_name:
            logo_img = load_and_prepare_image(f"./{logo_img_name}")

        with sv.VideoSink("./static/video_result.mp4", video_info=video_info) as sink:
            for frame_idx, object_ids, mask_logits in predictor.propagate_in_video(inference_state):
                if not os.path.exists(frames_paths[frame_idx]):
                    print(f"Warning: Frame '{frames_paths}' does not exist.")
                    continue

                frame = cv2.imread(frames_paths[frame_idx])
                if frame is None:
                    print(f"Warning: Could not read frame '{frames_paths}'.")
                    continue

                masks = (mask_logits > 0.0).cpu().numpy()
                N, X, H, W = masks.shape
                masks = masks.reshape(N * X, H, W)

                detections = sv.Detections(
                    xyxy=sv.mask_to_xyxy(masks=masks),
                    mask=masks,
                    tracker_id=np.array(object_ids)
                )

                result_frame = apply_masked_overlay(frame, masks, overlay_img)
                final_frame = mask_annotator.annotate(result_frame, detections)
                final_frame_with_logo = add_logo_to_frame(final_frame, logo_img, position='top-right')

                sink.write_frame(final_frame)

        input_video = "./static/video_result.mp4"
        output_video = "./static/output_compatible.mp4"

        audio_file = "./static/temp_audio.aac"
        audio_extracted = False
        try:
            audio_command = [
                "ffmpeg", "-i", f"./vid/{video_name}",
                "-vn", "-acodec", "aac", "-y", audio_file
            ]
            subprocess.run(audio_command, check=True)
            audio_extracted = True
        except subprocess.CalledProcessError as e:
            print(f"Audio extraction failed: {e}")

        if audio_extracted:
            command = [
                "ffmpeg", "-i", input_video, "-i", audio_file,
                "-vcodec", "libx264", "-acodec", "aac",
                "-strict", "-2", "-movflags", "+faststart",
                "-crf", "23", "-shortest", output_video
            ]
        else:
            command = [
                "ffmpeg", "-i", input_video,
                "-vcodec", "libx264", "-acodec", "aac",
                "-strict", "-2", "-movflags", "+faststart",
                "-crf", "23", output_video
            ]

        try:
            subprocess.run(command, check=True)
            print(f"Re-encoded video saved as: {output_video}")
        except subprocess.CalledProcessError as e:
            print(f"Error during re-encoding: {e}")

        # send the video to the frontend
        try:
            return send_file(
                output_video,
                mimetype='video/mp4',
                as_attachment=True,
                download_name='processed_video.mp4'
            )
        except Exception as e:
            return jsonify({"error": f"Error sending file: {str(e)}"}), 500

        return jsonify({
            "status": "success (no video sent)"
        })
    except Exception as e:
        return jsonify({"error": str(e), "status": "error"}), 500

if __name__ == "__main__":
    app.run(host="0.0.0.0", port=9000)
