#/bin/bash
killall lcnd
for i in 1 2 3 4
do
#	mkdir -p nohup
 	./bin/lcnd --config=$i.yaml &  # > nohup/$i.file 2>&1 &
	#./bin/lcnd --config=l0-ca-handshake/000${i}_abc/000${i}_abc.yaml &
done
