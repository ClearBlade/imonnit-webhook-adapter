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

#Clean up any old adapter stuff
echo "------Cleaning Up Old Adapter Configurations"
systemctl stop $ADAPTERSERVICENAME
systemctl disable $ADAPTERSERVICENAME
rm $SYSTEMDPATH/$ADAPTERSERVICENAME

echo "------Reloading daemon"
systemctl daemon-reload
