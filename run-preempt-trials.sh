#!/bin/bash

if [[ $# -lt 3 ]] || [[ $# -gt 8 ]]; then
  echo "Usage: PROGRAM GRACE CSV [HP] [MP] [FILLER] [CPU] [DELAY]"
  echo
  echo "Run PROGRAM (with PATH, ./program_name for current directory) with"
  echo "  GRACE number of seconds for termination grace period and"
  echo "  CSV true or false to write results to file in csv format"
  echo "  HP integer percent (0-100) of high-priority jobs"
  echo "  MP integer percent (0-100) of medium-priority jobs"
  echo "  FILLER number of background pods to start before test"
  echo "  CPU milli-cpus for each workload"
  echo "  DELAY seconds of delay between high priority workload starts"
  exit
fi

HP=0
MP=0
FILLER=0
CPU=100
DELAY=0

if [[ $# -ge 4 ]]; then
  HP=$4
fi

if [[ $# -ge 5 ]]; then
  MP=$5
fi

if [[ $# -ge 6 ]]; then
  FILLER=$6
fi

if [[ $# -ge 7 ]]; then
  CPU=$7
fi

if [[ $# -ge 8 ]]; then
  DELAY=$8
fi

trial_amounts=(50)
#trial_amounts=(5 10 15)
#trial_amounts=(5 10 15 20 30 40 50 75 100 200 300)
#trial_amounts=(50 75 100 200)
for i in "${trial_amounts[@]}"
do
  $1 --num=$i --grace=$2 --csv=$3 --hp=$HP --mp=$MP --filler=$FILLER --cpu=$CPU --delay=$DELAY
done
