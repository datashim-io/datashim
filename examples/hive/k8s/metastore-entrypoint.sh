#!/bin/bash

bin/hive --service schemaTool -dbType postgres -info 2> /dev/null

if [ $? -ne 0 ]; then    
   bin/hive --service schemaTool -initSchema -dbType postgres -verbose
fi

exec "$@"



