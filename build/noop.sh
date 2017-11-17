#/bin/bash
killall lcnd
mkdir -p nohup
./bin/lcnd --config=1.yaml > nohup/1.file 2>&1 &
