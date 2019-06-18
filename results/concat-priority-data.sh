#!/bin/bash

if [[ $# < 2 ]] || [[ $# > 3 ]]; then
    echo "Usage: $0 TEST_TYPE GRACE1 [GRACE2]"
    echo
    echo "  TEST_TYPE is one of k8s-api, k8s, docker"
    echo "  GRACE1 the grace value in the file names"
    echo "  GRACE2 the grace value in the file names"
    echo
    echo "Sequentially concatenates .csv files to create files ready to plot."
    echo "If GRACE2 is provided, the graph is composed of a first half and"
    echo "second half. Lower GRACE1 values, then higher GRACE2 values."
    echo "Otherwise, if no GRACE2 is provided, only the values for GRACE1 are"
    echo "concatenated."
    echo
    echo "Output file is named TEST_TYPE.csv"
    echo
    echo "Directory naming is expected to be of format TEST_TYPE-gGRACE1 or TEST_TYPE-gGRACE2"
    exit
fi

d1="${1}-g${2}"
c1="${d1}/g${2}-${1}-test.csv"
cp $d1/15*/*.csv $d1
paste -s -d'\n' $d1/n5-g$2-$1-test.csv $d1/n10-g$2-$1-test.csv $d1/n15-g$2-$1-test.csv > $c1

if [[ $# > 2 ]]; then
    d2="${1}-g${3}"
    c2="${d2}/g${3}-${1}-test.csv"
    cp $d2/15*/*.csv $d2
    paste -s -d'\n' $d2/n5-g$3-$1-test.csv $d2/n10-g$3-$1-test.csv $d2/n15-g$3-$1-test.csv > $c2

    paste -s -d'\n' $c1 $c2 > $1.csv
else
    mv $c1 $1.csv
fi
