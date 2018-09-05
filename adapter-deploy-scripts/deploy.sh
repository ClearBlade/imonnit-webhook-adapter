#!/bin/bash

if [ "$EUID" -ne 0 ]
  then
    echo "----------Permissions Error---------"
    echo "STOPPING: Please run as root or sudo"
    echo "------------------------------------"
  exit
fi

SCRIPTDIR="${0%/*}"
CONFIGFILENAME="adapterconfig.txt"
source "$SCRIPTDIR/$CONFIGFILENAME"

echo "Adapter Service Name: $ADAPTERSERVICENAME"
echo "Adapter Bin Path: $SCRIPTDIR/$ADAPTERBIN"
echo "SystemD Path: $SYSTEMDPATH"
echo "Platform URL: $PLATFORMURL"
echo "Messaging URL: $MESSAGINGURL"
echo "System Key: $SYSTEMKEY"
echo "System Secret: $SYSTEMSECRET"
echo "Device Name: $DEVICENAME"
echo "Active Key: $ACTIVEKEY"

chmod +x "./"

echo "-------Cleaning Up Old Adapter-------"
sudo systemctl stop $ADAPTERSERVICENAME
sudo systemctl disable $ADAPTERSERVICENAME
sudo rm $SYSTEMDPATH/$ADAPTERSERVICENAME
systemctl daemon-reload

echo "-------Configuring Adapter as a Service-------"
cat > "$SYSTEMDPATH/$ADAPTERSERVICENAME" <<EOF
[Unit]
Description=ClearBlade Monnit Webhook Adapter
After=network.target clearblade.service

[Service]
Type=simple
ExecStart=$(pwd)/$ADAPTERBIN -platformURL $PLATFORMURL -messagingURL $MESSAGINGURL -systemKey $SYSTEMKEY -systemSecret $SYSTEMSECRET -deviceName $DEVICENAME -activeKey $ACTIVEKEY -enableTLS -tlsCertPath /etc/ssl/clearblade.crt -tlsKeyPath /etc/ssl/clearblade.key -receiverPort 443
Restart=on-abort
TimeoutSec=30
RestartSec=30
StartLimitInterval=350
StartLimitBurst=10

[Install]
WantedBy=multi-user.target
EOF

echo "-------Reloading Daemon-------"
systemctl daemon-reload
echo "-------Enabling Startup on Reboot-------"
systemctl enable "$ADAPTERSERVICENAME"
systemctl start "$ADAPTERSERVICENAME"
echo "-------Adapter Deployed!-------"
