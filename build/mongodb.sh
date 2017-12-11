#!/bin/bash

dbpath="mongodb"
binPath="mongodb/bin/mongod"
dataPath="mongodb/data/db"

 if [ $# -ne 1 ]; then
     echo "eg: ./mongodb download or ./mongodb start or ./mongodb kill or ./mongodb clear"    
     exit        
 fi                     

action=$1

function downloadMongodb(){
    
    version_ubuntu1604=`uname -v | grep "16.04"`
    version_ubuntu1404=`uname -v | grep "14.04"`



    if [ ${#version_ubuntu1604} -ne 0 ];then
        echo version_ubuntu1604

        curl -O https://fastdl.mongodb.org/linux/mongodb-linux-x86_64-ubuntu1604-3.4.10.tgz

        tar -zxvf mongodb-linux-x86_64-ubuntu1604-3.4.10.tgz

        mv mongodb-linux-x86_64-ubuntu1604-3.4.10 $dbpath

        return
    fi

    if [ ${#version_ubuntu1404} -ne 0 ];then
        echo version_ubuntu1604        
        #curl -O https://fastdl.mongodb.org/linux/mongodb-linux-x86_64-ubuntu1404-3.4.10.tgz
    
        tar -zxvf mongodb-linux-x86_64-ubuntu1404-3.4.10.tgz

        mv mongodb-linux-x86_64-ubuntu1404-3.4.10 $dbpath

        return
    fi
}

function startMongodb(){
    if [ ! -f $binPath ];then
        echo "bin file not exist"
    else
        if [ ! -d $dataPath ];then
            echo "data path is nil make data folder"
            mkdir -p $dataPath
        fi 
        ./$binPath --dbpath $dataPath &     
    fi
}

function killMongodb(){
    ps -aux | awk '/mongod/ {print $2}'|xargs kill -9
}

function clearMongodb(){
    if [ -d $dataPath ];then
        rm -r $dataPath
    fi        
}



 if [ $action == "download" ]; then
    echo "download mongodb..."    
    downloadMongodb
 elif [ $action == "start" ];then
    echo "start mongodb..."    
    startMongodb
 elif [ $action == "kill" ];then
    echo "kill mongodb..."        
     killMongodb
 elif [ $action == "clear" ];then
    echo "clear mongodb..."         
     clearMongodb
 else
    echo "not spport command ,eg: download or start or kill or clear"  
    exit  
 fi










