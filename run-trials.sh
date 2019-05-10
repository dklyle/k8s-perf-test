#!/bin/bash

if [[ $# != 3 ]]; then
  echo "Usage: $0 PROGRAM GRACE CSV"
  echo
  echo "Run PROGRAM (with PATH, ./program_name for current directory) with"
  echo "  GRACE number of seconds for termination grace period and"
  echo "  CSV true or false to write results to file in csv format"
  exit
fi

trial_amounts=(5 10 15 20 30 40 50 75 100)
for i in "${trial_amounts[@]}"
do
  $1 --num=$i --grace=$2 --csv=$3
done
