#!/bin/bash
JOBS=
for i in {1..50}
do
go run main.go -addr=:9998 &
JOBS="${JOBS} $!"
done

echo $JOBS > pids.txt
sleep 60s
for JOB in $JOBS
do
kill -9 $JOB
done
