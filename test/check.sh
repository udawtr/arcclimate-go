#!/bin/bash

modes=("normal" "EA")
mode_separates=("Nagata" "Watanabe" "Erbs" "Udagawa" "Perez")

for mode in "${modes[@]}"; do
  for mode_separate in "${mode_separates[@]}"; do
    filename1="${mode}_${mode_separate,,}.csv"
    filename2="${mode}_${mode_separate,,}_go.csv"
    echo "Check $filename1 and $filename2"
    python3 diff.py "$filename1" "$filename2"

    if [ $? -ne 0 ]; then
      echo "Error occurred while running diff.py. Exiting."
      exit 1
    fi
  done
done
