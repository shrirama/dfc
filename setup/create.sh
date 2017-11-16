#!/bin/bash

INSTANCEPREFIX="dfc"
TMPDIR="/tmp/nvidia"
CACHEDIR="cache"
LOGDIR="log"
PROXYURL="http://localhost:8080"

# Starting Portnumber
PORT=8079

# Starting ID
ID=0

PROTO="tcp"
CLDPROVIDER="amazon"
DIRPATH="/tmp/nvidia/"
CACHEDIR="/cache"
LOGDIR="/log"
CONFPATH="/etc/dfconf"
INSTANCEPREFIX="dfc"
MAXCONCURRENTDOWNLOAD=64
MAXCONCURRENTUPLOAD=64
MAXPARTSIZE=4294967296


echo Enter Total Number of Proxy + Storage Server to be started.
echo There will be 1 Proxy server and Rest would be storage servers.
read servcount
START=0
END=$servcount

for (( c=$START; c<$END; c++ ))
do
		ID=$(expr $ID + 1)
		PORT=$(expr $PORT + 1)
		CURINSTANCE=$INSTANCEPREFIX$c
		CONFFILE=$CONFPATH$c.json
		cat > $CONFFILE <<EOL
		[
			{
				"proto": "${PROTO}",
				"port":	"${PORT}",
				"id": "${ID}",
				"proxyclienturl": "${PROXYURL}",
				"cachedir":	"${DIRPATH}${CURINSTANCE}${CACHEDIR}",
				"logdir":	"${DIRPATH}${CURINSTANCE}${LOGDIR}",
				"cloudprovider":	"${CLDPROVIDER}",
				"maxconcurrdownld":	${MAXCONCURRENTDOWNLOAD},
				"maxconcurrupld":	${MAXCONCURRENTUPLOAD},
				"maxpartsize":	${MAXPARTSIZE}	
			}
		]
EOL
done

# Start Proxy Client and Storage Daemon, First Configuration file is used for Proxy 
# and subsequent one is used for Storage Server(s).
for (( c=$START; c<$END; c++ ))
do
		CONFFILE=$CONFPATH$c.json
		if [ $c -eq 0 ]
		then
				go run dfcstart.go -configfile=$CONFFILE -type=proxy &
		else
				go run dfcstart.go -configfile=$CONFFILE -type=server &
		fi
done
