#!/bin/bash

mkdir ../sam2seg/frames/$1

ffmpeg -i ../sam2seg/vid/$1.mp4 -q:v 2 -start_number 0 ../sam2seg/frames/$1/'%05d.jpg'
