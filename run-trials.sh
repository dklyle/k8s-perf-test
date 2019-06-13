#!/bin/bash

if [[ $# -lt 3 ]] || [[ $# -gt 5 ]]; then
  echo "Usage: PROGRAM GRACE CSV [HP] [MP]"
  echo
  echo "Run PROGRAM (with PATH, ./program_name for current directory) with"
  echo "  GRACE number of seconds for termination grace period and"
  echo "  CSV true or false to write results to file in csv format"
  echo "  HP integer percent (0-100) of high-priority jobs"
  echo "  MP integer percent (0-100) of medium-priority jobs"
  exit
fi

HP=0
MP=0

if [[ $# -ge 4 ]]; then
  HP=$4
fi

if [[ $# -ge 5 ]]; then
  MP=$5
fi

#trial_amounts=(5 10 15 20 30)
trial_amounts=(5 10 15 20 30 40 50 75 100 200 300)
#trial_amounts=(50 75 100 200)
for i in "${trial_amounts[@]}"
do
  $1 --num=$i --grace=$2 --csv=$3 --hp=$HP --mp=$MP
done
