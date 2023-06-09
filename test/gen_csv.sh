#!/bin/bash

modes=("normal" "EA")
mode_separates=("Nagata" "Watanabe" "Erbs" "Udagawa" "Perez")

for mode in "${modes[@]}"; do
  for mode_separate in "${mode_separates[@]}"; do
    filename="${mode}_${mode_separate,,}_go.csv"
    ../arcclimate-go 36.1290111 140.0754174 --mode "$mode" --mode_separate "$mode_separate" -o "$filename"
    echo "Output file: $filename"
  done
done
