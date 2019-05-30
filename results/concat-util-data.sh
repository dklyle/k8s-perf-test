#!/bin/bash

if [[ $# < 3 ]]; then
    echo "Usage: $0 TEST_TYPE GRACE NODE"
    echo
    echo "  TEST_TYPE is one of k8s-api, k8s, docker"
    echo "  GRACE the grace value in the file names"
    echo "  NODE(s) are the list of node names to concat data for."
    echo
    echo "Sequentially concatenates .csv files to create files ready to plot."
    echo
    echo "Output file is named TEST_TYPE-util.csv"
    echo
    echo "Directory naming is expected to be of format TEST_TYPE-gGRACE1 or TEST_TYPE-gGRACE2"
    exit
fi

d1="${1}-g${2}"
cp $d1/node*-155*/*.csv $d1
i=3
while [[ $i -le $# ]]; do
    node=${@:$i:1}
    echo $node
    c1="${d1}/g${2}-${1}-test-util-${node}.csv"
    paste -s -d'\n' $d1/n5-g$2-$node-$1-test-util.csv $d1/n10-g$2-$node-$1-test-util.csv $d1/n15-g$2-$node-$1-test-util.csv $d1/n20-g$2-$node-$1-test-util.csv $d1/n30-g$2-$node-$1-test-util.csv $d1/n40-g$2-$node-$1-test-util.csv $d1/n50-g$2-$node-$1-test-util.csv $d1/n75-g$2-$node-$1-test-util.csv $d1/n100-g$2-$node-$1-test-util.csv $d1/n200-g$2-$node-$1-test-util.csv > $c1
    mv $c1 $1-util-$node.csv
    i=$i+1
done

