#!/bin/bash

# kill all withdraw_order
ps x | grep withdraw_order | awk '{print $1}' | xargs kill >null 2>&1
rm -f null

# start lcnd
for i in 1 2
do
	mkdir -p nohup
 	./withdraw_order -atomic=3 -withdraw=3 -order=3 > nohup/$i.file 2>&1 &
done
