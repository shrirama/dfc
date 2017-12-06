#!/bin/bash

INSTANCEPREFIX="dfc"
TMPDIR="/tmp/nvidia"
CACHEDIR="cache"
LOGDIR="log"
PROXYURL="http://localhost:8080"
PASSTHRU=true

# Starting Portnumber
PORT=8079

# Starting ID
ID=0

PROTO="tcp"
CLDPROVIDER="amazon"
DIRPATH="/tmp/nvidia/"
CACHEDIR="/cache"
LOGLEVEL="3"
LOGDIR="/log"
CONFPATH="/etc/dfconf"
INSTANCEPREFIX="dfc"
MAXCONCURRENTDOWNLOAD=64
MAXCONCURRENTUPLOAD=64
MAXPARTSIZE=4294967296


echo Enter number of caching servers:
read servcount
START=0
END=$servcount

for (( c=$START; c<=$END; c++ ))
do
	ID=$(expr $ID + 1)
	PORT=$(expr $PORT + 1)
	CURINSTANCE=$INSTANCEPREFIX$c
	CONFFILE=$CONFPATH$c.json
	cat > $CONFFILE <<EOL
	{
		"id": 				"${ID}",
		"cachedir":			"${DIRPATH}${CURINSTANCE}${CACHEDIR}",
		"logdir":			"${DIRPATH}${CURINSTANCE}${LOGDIR}",
		"loglevel": 			"${LOGLEVEL}",
		"cloudprovider":		"${CLDPROVIDER}",
		"listen": {
			"proto": 		"${PROTO}",
			"port":			"${PORT}"
		},
		"proxy": {
			"url": 			"${PROXYURL}",
			"passthru": 		${PASSTHRU}
		},
		"s3": {
			"maxconcurrdownld":	${MAXCONCURRENTDOWNLOAD},
			"maxconcurrupld":	${MAXCONCURRENTUPLOAD},
			"maxpartsize":		${MAXPARTSIZE}	
		}
	}
EOL
done

# Start Proxy Client and Storage Daemon, First Configuration file is used for Proxy 
# and subsequent one is used for Storage Server(s).
for (( c=$START; c<=$END; c++ ))
do
		CONFFILE=$CONFPATH$c.json
		if [ $c -eq 0 ]
		then
				go run setup/dfcstart.go -configfile=$CONFFILE -type=proxy &
#Need to wait for Proxy Client to be ready to accept new connections
				sleep 3
		else
				go run setup/dfcstart.go -configfile=$CONFFILE -type=server &
		fi
done
