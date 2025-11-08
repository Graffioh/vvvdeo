import os
os.environ["PYTORCH_ENABLE_MPS_FALLBACK"] = "1"
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
import torch
from sam2.sam2_video_predictor import SAM2VideoPredictor

app = Flask(__name__)

# logging setup
log_formatter = logging.Formatter(
    '%(asctime)s - %(levelname)s - %(message)s [in %(pathname)s:%(lineno)d]'
)
log_handler = RotatingFileHandler('app.log', maxBytes=1000000, backupCount=3)
log_handler.setFormatter(log_formatter)
log_handler.setLevel(logging.DEBUG)
app.logger.addHandler(log_handler)
app.logger.setLevel(logging.DEBUG)
app.logger.info("Starting the application...")

if torch.cuda.is_available():
    device = torch.device("cuda")
#elif torch.backends.mps.is_available():
#    device = torch.device("mps")
else:
    device = torch.device("cpu")
print(f"using device: {device}")

if device.type == "cuda":
    # use bfloat16 for the entire notebook
    torch.autocast("cuda", dtype=torch.bfloat16).__enter__()
    # turn on tfloat32 for Ampere GPUs (https://pytorch.org/docs/stable/notes/cuda.html#tensorfloat-32-tf32-on-ampere-devices)
    if torch.cuda.get_device_properties(0).major >= 8:
        torch.backends.cuda.matmul.allow_tf32 = True
        torch.backends.cudnn.allow_tf32 = True
elif device.type == "mps":
    print(
        "\nSupport for MPS devices is preliminary. SAM 2 is trained with CUDA and might "
        "give numerically different outputs and sometimes degraded performance on MPS. "
        "See e.g. https://github.com/pytorch/pytorch/issues/84936 for a discussion."
    )

# local model
#sam2_checkpoint = "./src/sam-2/checkpoints/sam2.1_hiera_small.pt"
#model_cfg = "./configs/sam2.1/sam2.1_hiera_s.yaml"
#predictor = build_sam2_video_predictor(model_cfg, sam2_checkpoint, device=device)

# Hugging Face model
model_str = "facebook/sam2-hiera-base-plus" if torch.cuda.is_available() else "facebook/sam2-hiera-tiny"
predictor = SAM2VideoPredictor.from_pretrained(model_str, device=device)

# def get_path_from_presignedurl(key) -> str:
#         url = f"http://localhost:8080/presigned-url/get?key={key}"
#         try:
#             response = requests.get(url)
#             response.raise_for_status()
#             response_data = response.json()

#             path = response_data.get('presignedUrl')
#             if not path:
#                 app.logger.error("Response did not contain presignedUrl!")
#                 raise RuntimeError("Response did not contain 'presignedUrl'.")
#             return path
#         except requests.RequestException as e:
#             app.logger.error("Failed to fetch presigned URL!")
#             raise RuntimeError(f"Failed to fetch presigned URL: {e}")

# def download_video(video_url, temp_dir):
#     local_video_path = os.path.join(temp_dir, "downloaded_video.mp4")

#     try:
#         with requests.get(video_url, stream=True) as response:
#             response.raise_for_status()
#             with open(local_video_path, 'wb') as f:
#                 shutil.copyfileobj(response.raw, f)
#     except requests.exceptions.RequestException as e:
#         app.logger.exception("Failed to download the original video")
#         raise ValueError(f"Failed to download video: {e}")

#     return local_video_path

# def download_and_extract_zip(zip_url, temp_dir):
#     zip_path = os.path.join(temp_dir, "downloaded_archive.zip")
#     extract_dir = os.path.join(temp_dir, "frames")

#     try:
#         with requests.get(zip_url, stream=True) as response:
#             response.raise_for_status()
#             with open(zip_path, 'wb') as f:
#                 shutil.copyfileobj(response.raw, f)
#     except requests.exceptions.RequestException as e:
#         app.logger.exception("Failed to download zip file from r2 storage!")
#         raise ValueError(f"Failed to download zip file: {e}")

#     try:
#         with zipfile.ZipFile(zip_path, 'r') as zip_ref:
#             zip_ref.extractall(extract_dir)
#     except zipfile.BadZipFile:
#         app.logger.exception("The downloaded file is not a valid .zip archive")
#         raise ValueError("The downloaded file is not a valid .zip archive")

#     app.logger.info("Zip extracted sucessfully")

#     return extract_dir

def load_and_prepare_image(image_path, required_format="RGBA"):
    if not os.path.exists(image_path):
        app.logger.error(f"Image file not found: {image_path}.")
        raise ValueError(f"Image file not found: {image_path}")

    image = cv2.imread(image_path, cv2.IMREAD_UNCHANGED)
    if image is None:
        app.logger.error(f"Failed to load image: {image_path}.")
        raise ValueError(f"Failed to load image: {image_path}")

    if len(image.shape) == 2:
        image = cv2.cvtColor(image, cv2.COLOR_GRAY2RGB)

    if required_format == "RGBA":
        if len(image.shape) == 3 and image.shape[2] == 3:
            alpha_channel = np.ones(image.shape[:2], dtype=image.dtype) * 255
            image = cv2.merge([image, alpha_channel])
        elif len(image.shape) != 3 or image.shape[2] != 4:
            app.logger.error("Image is not in RGB or RGBA format!")
            raise ValueError(f"Image must be in RGB or RGBA format. Current shape: {image.shape}")

    return image

def apply_image_to_segmentation(frame, masks, overlay_img):
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

def prepare_overlay_img(overlay_img_file):
    if not overlay_img_file:
        app.logger.exception("Overlay image file not found!")
        raise ValueError("Overlay image file not found!")

    file_bytes = np.frombuffer(overlay_img_file.read(), np.uint8)
    overlay_img = cv2.imdecode(file_bytes, cv2.IMREAD_UNCHANGED)
    if overlay_img is not None and overlay_img.shape[-1] != 4:
        b, g, r = cv2.split(overlay_img)[:3]
        alpha = np.full((overlay_img.shape[0], overlay_img.shape[1]), 255, dtype=np.uint8)
        overlay_img = cv2.merge((b, g, r, alpha))

    return overlay_img

def prepare_logo_img():
    logo_img = None
    logo_img_name = "vvvdeo-logo.png"
    if logo_img_name:
        logo_img = load_and_prepare_image(f"./{logo_img_name}")
        app.logger.info("Logo image loaded and prepared.")

    return logo_img

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

def reencode_audio_in_video(temp_dir, local_video_path):
    input_video = temp_dir + "/video_result.mp4"
    output_video = temp_dir + "/output_compatible.mp4"

    audio_file = temp_dir + "/temp_audio.aac"
    audio_extracted = False
    try:
        audio_command = [
            "ffmpeg", "-i", local_video_path,
            "-vn", "-acodec", "aac", "-y", audio_file
        ]
        subprocess.run(audio_command, check=True)
        audio_extracted = True
    except subprocess.CalledProcessError as e:
        app.logger.exception("Audio extraction from original video failed.")
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
        app.logger.info("Video successfully re-encoded")
    except subprocess.CalledProcessError as e:
        app.logger.exception("Error during video re-encoding!")

    return output_video

def propagate_and_sink_in_video(inference_state, temp_dir, video_info, frames_paths, overlay_img, logo_img):
    with sv.VideoSink(temp_dir + "/video_result.mp4", video_info=video_info) as sink:
        for frame_idx, object_ids, mask_logits in predictor.propagate_in_video(inference_state):
            if not os.path.exists(frames_paths[frame_idx]):
                app.logger.warning(f"Frame '{frames_paths[frame_idx]}' does not exist.")
                continue

            # read the frame
            frame = cv2.imread(frames_paths[frame_idx])
            if frame is None:
                app.logger.warning("Frame processed is 'None'")
                continue

            # convert mask_logits (result tensors from inference) into masks (binary mask, 1 foreground or 0 background)
            masks = (mask_logits > 0.0).cpu().numpy()

            # optimize the masks by reshaping them for subsequent processing
            N, X, H, W = masks.shape
            masks = masks.reshape(N * X, H, W)

            # apply selected image into the segment by using image manipulation magic
            frame_with_image = apply_image_to_segmentation(frame, masks, overlay_img)
            final_frame_with_logo = add_logo_to_frame(frame_with_image, logo_img, position='top-right')

            # combine the frame with the others frames to create the final modified video
            sink.write_frame(final_frame_with_logo)


@app.route("/segment", methods=["POST"])
def segment():
    try:
        with tempfile.TemporaryDirectory() as temp_dir:
            app.logger.debug("Temp directory created: %s", temp_dir)

            # get r2 bucket object key
            # file_key = request.form.get('fileKey')
            # if not file_key:
            #     app.logger.error("Missing 'file_key' in request body")
            #     return jsonify({"error": "Missing 'file_key' in request body"}), 400
            # video_name = file_key

            # fetch video and frames with presigned urls
            # cloudflare_video_path = get_path_from_presignedurl("videos/" + file_key)
            # local_video_path = download_video(cloudflare_video_path, temp_dir)
            #
            # cloudflare_frames_path = get_path_from_presignedurl("frames/" + file_key + ".zip")
            # local_frames_path = download_and_extract_zip(cloudflare_frames_path, temp_dir)

            video_name = "to_segment.mp4"
            local_video_path = "./video/" + video_name
            local_frames_path = "./frames/"
            frames_paths = sorted(sv.list_files_with_extensions(directory=local_frames_path, extensions=["jpg"]))

            try:
                video_info = sv.VideoInfo.from_video_path(local_video_path)
                app.logger.debug("Video info successfully retrieved")
            except AttributeError as e:
                app.logger.exception("Invalid video name format")
                raise ValueError("Invalid video name format") from e
            except FileNotFoundError as e:
                app.logger.exception(f"Video file not found: {video_name}")
                raise ValueError(f"Video file not found: {video_name}") from e

            # inference (initial state)
            app.logger.debug("Initializing SAM2 predictor with frames path: %s", local_frames_path)
            try:
                inference_state = predictor.init_state(video_path=local_frames_path)
                predictor.reset_state(inference_state)
                app.logger.info("Inference state initialized and reset")
            except Exception as e:
                app.logger.exception(f"Failed to initialize or reset SAM2 predictor: {str(e)}")
                return jsonify({"error": f"Failed to initialize SAM2 predictor: {str(e)}"}), 500

            # get points and labels from frontend request
            segmentation_data = request.form.get("segmentationData")
            if not segmentation_data:
                app.logger.error("Missing 'segmentationData' in request body")
                return jsonify({"error": "Missing 'segmentationData' in request body"}), 400

            try:
                segmentation_data_json = json.loads(segmentation_data)
                app.logger.debug("Segmentation Data received as JSON: %s", segmentation_data_json)
            except json.JSONDecodeError as e:
                app.logger.error("Error decoding segmentationData JSON: %s", str(e))
                return jsonify({"error": "Invalid JSON in 'segmentationData'"}), 400

            points = segmentation_data_json.get("coordinates", [])
            labels = segmentation_data_json.get("labels", [])
            if not points:
                app.logger.error("Missing coordinates!")
                return jsonify({"error": "Missing coordinates"}), 400

            try:
                points = np.array([[p['x'], p['y']] for p in points], dtype=np.float32)
                labels = np.array([l for l in labels], dtype=np.int32)
            except Exception as e:
                app.logger.exception("Error processing points")
                return jsonify({"error": f"Error processing points: {str(e)}"}), 500

            # add points used for the segmentation
            segmentation_frame_idx = 0
            _, _, _ = predictor.add_new_points_or_box(
                inference_state=inference_state,
                frame_idx=segmentation_frame_idx,
                obj_id=1,
                points=points,
                labels=labels,
            )

            # prepare the image to apply on top of the segmentation
            overlay_img_file = request.files.get("image")
            overlay_img = prepare_overlay_img(overlay_img_file)
            logo_img = prepare_logo_img()

            # propagate the mask to all the frames and sink them into one video
            app.logger.info("Video propagation and sinking starting...")
            propagate_and_sink_in_video(inference_state, temp_dir, video_info, frames_paths, overlay_img, logo_img)
            app.logger.info("Video propagation successful.")

            # fix audio not working by re-encoding the audio directly in the video
            output_video = reencode_audio_in_video(temp_dir, local_video_path)

            # send the video to the frontend for download
            try:
                return send_file(
                    output_video,
                    mimetype='video/mp4',
                    as_attachment=True,
                    download_name='crafted_vvvdeo.mp4'
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
