#!/bin/bash

# kill all lcnd and vm
ps x | grep lcnd | awk '{print $1}' | xargs kill >null 2>&1
rm -f null

# start lcnd
for i in 1 # 2 3 4 # 5
do
	mkdir -p nohup
 	./bin/lcnd --config=$i.yaml > nohup/$i.file 2>&1 &
	#./bin/lcnd --config=l0-ca-handshake/000${i}_abc/000${i}_abc.yaml &
done
