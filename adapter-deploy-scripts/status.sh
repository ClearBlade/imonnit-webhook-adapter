#!/bin/bash
if [ "$EUID" -ne 0 ]
  then 
  	echo "---------Permissions Error---------"
  	echo "STOPPING: Please run as root or sudo"
  	echo "-----------------------------------"
  exit
fi

SCRIPTDIR="${0%/*}"
CONFIGFILENAME="adapterconfig.txt"
source "$SCRIPTDIR/$CONFIGFILENAME"
# If this script is executed, we know the adapter has been deployed. No need to test for that.
STATUS="Deployed"


PID=$(ps aux | grep -v grep | grep $ADAPTERBIN | awk '{print $2}')
#echo $PID
if [[ $PID ]]; then
	STATUS="Running"
else
	STATUS="Stopped"
fi

echo $STATUS